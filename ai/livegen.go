package ai

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/anthonynsimon/bild/transform"

	"github.com/SvenDH/go-card-engine/ui"
)

type LiveGenerator struct {
	W, H int
	
	// Source image
	sourceImg image.Image
	
	// Output tilemap (updated progressively)
	tilemap *ui.TileMap
	
	// Generation state
	totalCells   int
	completed    atomic.Int32
	generating   bool
	done         bool
	maxIter      int     // Maximum iterations per cell (0 = unlimited)
	maxColors    int     // Maximum palette colors to use per cell (0 = all)
	edgeWeight   float32 // Weight for edge alignment in error (0 = disabled)
	
	// Pre-computed data
	tilesPix     [][]color.NRGBA
	tilesA       [][]uint8
	tilesEdges   [][]float32 // Edge magnitude for each tile
	srcPix       []color.NRGBA
	srcEdges     []float32   // Edge magnitude for source
	bgTileIdx    int
	tileCount    int
	tileOrder    []int // Randomized tile order
	
	// Optimization caches
	tileAvgR     []int
	tileAvgG     []int
	tileAvgB     []int
	paletteRGB   []struct{ R, G, B int }
	
	// AI embeddings for semantic similarity
	onnxEmbedder       *ONNXEmbedder
	onnxModelPath      string
	
	// CLIP text-guided generation
	clipModel          *CLIPModel
	clipPrompt         string
	clipEmbedding      []float32 // Encoded prompt
	
	// Evolutionary algorithm settings
	evolutionPopSize   int
	evolutionGens      int
	
	mu sync.RWMutex
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func NewLiveGenerator(width, height int, src image.Image) *LiveGenerator {
	return &LiveGenerator{
		W:          width,
		H:          height,
		sourceImg:  src,
		tilemap:    ui.NewTileMap(width, height, nil),
		totalCells: width * height,
		maxIter:    0,   // 0 = unlimited (exhaustive search)
		maxColors:  0,   // 0 = use all colors
		edgeWeight: 0.0, // 0 = disabled by default
	}
}

// gaussianBlur applies a simple 3x3 Gaussian blur
func gaussianBlur(pixels []color.NRGBA, width, height int) []color.NRGBA {
	blurred := make([]color.NRGBA, len(pixels))
	copy(blurred, pixels)
	
	// Gaussian 3x3 kernel (approximation): [1 2 1; 2 4 2; 1 2 1] / 16
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			idx := y*width + x
			
			sumR, sumG, sumB := 0, 0, 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					n := (y+dy)*width + (x+dx)
					p := pixels[n]
					
					// Gaussian weights
					weight := 1
					if dx == 0 && dy == 0 {
						weight = 4
					} else if dx == 0 || dy == 0 {
						weight = 2
					}
					
					sumR += int(p.R) * weight
					sumG += int(p.G) * weight
					sumB += int(p.B) * weight
				}
			}
			
			blurred[idx] = color.NRGBA{
				R: uint8(sumR / 16),
				G: uint8(sumG / 16),
				B: uint8(sumB / 16),
				A: pixels[idx].A,
			}
		}
	}
	
	return blurred
}

// computeEdges calculates edge magnitude using Sobel operator
func computeEdges(pixels []color.NRGBA, width, height int) []float32 {
	// Apply Gaussian blur first to reduce noise
	blurred := gaussianBlur(pixels, width, height)
	
	edges := make([]float32, len(pixels))
	
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			idx := y*width + x
			
			// Sobel kernels for horizontal and vertical gradients
			// Gx = [-1 0 1; -2 0 2; -1 0 1]
			// Gy = [-1 -2 -1; 0 0 0; 1 2 1]
			
			// Get 3x3 neighborhood grayscale values from blurred image
			gray := make([]float32, 9)
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					n := (y+dy)*width + (x+dx)
					p := blurred[n]
					// Convert to grayscale
					gray[(dy+1)*3+(dx+1)] = float32(p.R)*0.299 + float32(p.G)*0.587 + float32(p.B)*0.114
				}
			}
			
			// Apply Sobel
			gx := -gray[0] + gray[2] - 2*gray[3] + 2*gray[5] - gray[6] + gray[8]
			gy := -gray[0] - 2*gray[1] - gray[2] + gray[6] + 2*gray[7] + gray[8]
			
			// Edge magnitude
			edges[idx] = float32(gx*gx + gy*gy) // Use squared magnitude to avoid sqrt
		}
	}
	
	return edges
}

func (g *LiveGenerator) SetMaxIterations(maxIter int) {
	g.maxIter = maxIter
}

func (g *LiveGenerator) SetMaxColors(maxColors int) {
	g.maxColors = maxColors
}

func (g *LiveGenerator) SetEdgeWeight(weight float32) {
	g.edgeWeight = weight
}

func (g *LiveGenerator) SetONNXModel(modelPath string) {
	g.onnxModelPath = modelPath
}

func (g *LiveGenerator) EnableEvolution(popSize, generations int) {
	g.evolutionPopSize = popSize
	g.evolutionGens = generations
}

func (g *LiveGenerator) EnableCLIP(prompt, textModel, visionModel, tokenizerPath string) error {
	// Load CLIP model
	clipModel, err := NewCLIPModel(textModel, visionModel, tokenizerPath)
	if err != nil {
		return fmt.Errorf("failed to load CLIP model: %w", err)
	}
	
	// Encode the prompt
	embedding, err := clipModel.EncodeText(prompt)
	if err != nil {
		clipModel.Close()
		return fmt.Errorf("failed to encode prompt: %w", err)
	}
	
	g.clipModel = clipModel
	g.clipPrompt = prompt
	g.clipEmbedding = embedding
	
	return nil
}

func (g *LiveGenerator) Init() ui.Cmd {
	if g.sourceImg == nil || ui.TilesSource == nil || len(ui.PaletteRGBA) == 0 {
		g.done = true
		return nil
	}
	
	// Start generation in background
	go g.generate()
	g.generating = true
	
	return nil
}

func (g *LiveGenerator) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return g, nil
}

func (g *LiveGenerator) View() *ui.TileMap {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.tilemap
}

func (g *LiveGenerator) Progress() float64 {
	if g.totalCells == 0 {
		return 1.0
	}
	return float64(g.completed.Load()) / float64(g.totalCells)
}

func (g *LiveGenerator) IsDone() bool {
	return g.done
}

func (g *LiveGenerator) generate() {
	// Prepare tiles
	tw := ui.TilesSource.Bounds().Dx() / ui.TileSize
	th := ui.TilesSource.Bounds().Dy() / ui.TileSize
	g.tileCount = tw * th
	if g.tileCount == 0 {
		g.done = true
		return
	}
	g.bgTileIdx = g.tileCount - 1
	
	// Resize source image with preprocessing for low-detail tilemaps
	targetW := g.W * ui.TileSize
	targetH := g.H * ui.TileSize
	
	// For very small grids (< 10x10), apply sharpening and contrast enhancement
	var src image.Image
	if g.W < 10 || g.H < 10 {
		// First downsample aggressively to simplify details
		intermediateW := g.W * 32  // Lower intermediate resolution
		intermediateH := g.H * 32
		intermediate := transform.Resize(g.sourceImg, intermediateW, intermediateH, transform.Lanczos)
		
		// Then upsample with sharpening
		src = transform.Resize(intermediate, targetW, targetH, transform.Linear)
	} else {
		// Normal resize for larger grids
		src = transform.Resize(g.sourceImg, targetW, targetH, transform.Lanczos)
	}
	
	pixelsPerTile := ui.TileSize * ui.TileSize
	palLen := len(ui.PaletteRGBA)
	
	// Pre-extract all tile pixels and compute averages + edges
	g.tilesPix = make([][]color.NRGBA, g.tileCount)
	g.tilesA = make([][]uint8, g.tileCount)
	g.tilesEdges = make([][]float32, g.tileCount)
	g.tileAvgR = make([]int, g.tileCount)
	g.tileAvgG = make([]int, g.tileCount)
	g.tileAvgB = make([]int, g.tileCount)
	
	for ti := 0; ti < g.tileCount; ti++ {
		sx := (ti % tw) * ui.TileSize
		sy := (ti / tw) * ui.TileSize
		g.tilesPix[ti] = make([]color.NRGBA, pixelsPerTile)
		g.tilesA[ti] = make([]uint8, pixelsPerTile)
		sumR, sumG, sumB := 0, 0, 0
		idx := 0
		for y := 0; y < ui.TileSize; y++ {
			for x := 0; x < ui.TileSize; x++ {
				c := color.NRGBAModel.Convert(ui.TilesSource.At(sx+x, sy+y)).(color.NRGBA)
				g.tilesPix[ti][idx] = c
				g.tilesA[ti][idx] = c.A
				sumR += int(c.R)
				sumG += int(c.G)
				sumB += int(c.B)
				idx++
			}
		}
		g.tileAvgR[ti] = sumR / pixelsPerTile
		g.tileAvgG[ti] = sumG / pixelsPerTile
		g.tileAvgB[ti] = sumB / pixelsPerTile
	}
	
	// Only compute edges if edge detection is enabled
	if g.edgeWeight > 0 {
		for ti := 0; ti < g.tileCount; ti++ {
			g.tilesEdges[ti] = computeEdges(g.tilesPix[ti], ui.TileSize, ui.TileSize)
		}
	}
	
	// Compute AI embeddings if enabled
	var err error
	
	log.Printf("Loading ONNX model from %s...", g.onnxModelPath)
	g.onnxEmbedder, err = NewONNXEmbedder(g.onnxModelPath)
	if err != nil {
		log.Printf("Failed to load ONNX model: %v", err)
	}
	
	// Pre-convert palette to int RGB for faster access
	g.paletteRGB = make([]struct{ R, G, B int }, palLen)
	for i := 0; i < palLen; i++ {
		g.paletteRGB[i].R = int(ui.PaletteRGBA[i].R)
		g.paletteRGB[i].G = int(ui.PaletteRGBA[i].G)
		g.paletteRGB[i].B = int(ui.PaletteRGBA[i].B)
	}
	
	// Create randomized tile order
	g.tileOrder = make([]int, g.tileCount)
	for i := 0; i < g.tileCount; i++ {
		g.tileOrder[i] = i
	}
	rand.Shuffle(g.tileCount, func(i, j int) {
		g.tileOrder[i], g.tileOrder[j] = g.tileOrder[j], g.tileOrder[i]
	})
	
	// Pre-extract source pixels and compute global average
	g.srcPix = make([]color.NRGBA, g.W*g.H*ui.TileSize*ui.TileSize)
	globalAvgR, globalAvgG, globalAvgB := 0, 0, 0
	totalPixels := 0
	
	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			base := (gy*g.W + gx) * ui.TileSize * ui.TileSize
			idx := 0
			for py := 0; py < ui.TileSize; py++ {
				for px := 0; px < ui.TileSize; px++ {
					x := gx*ui.TileSize + px
					y := gy*ui.TileSize + py
					c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
					g.srcPix[base+idx] = c
					globalAvgR += int(c.R)
					globalAvgG += int(c.G)
					globalAvgB += int(c.B)
					totalPixels++
					idx++
				}
			}
		}
	}
	
	globalAvgR /= totalPixels
	globalAvgG /= totalPixels
	globalAvgB /= totalPixels
	
	// Select colors based on actual usage frequency in the image
	type colorMatch struct {
		idx       int
		frequency int // How many pixels map to this color
	}
	
	// Count how many source pixels would map to each palette color
	globalColorMatches := make([]colorMatch, palLen)
	for i := 0; i < palLen; i++ {
		globalColorMatches[i].idx = i
		globalColorMatches[i].frequency = 0
	}
	
	// For each pixel, find its closest palette color and increment count
	for i := 0; i < len(g.srcPix); i++ {
		s := g.srcPix[i]
		bestDist := 999999
		bestIdx := 0
		
		for j := 0; j < palLen; j++ {
			dist := abs(g.paletteRGB[j].R-int(s.R)) + abs(g.paletteRGB[j].G-int(s.G)) + abs(g.paletteRGB[j].B-int(s.B))
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}
		
		globalColorMatches[bestIdx].frequency++
	}
	
	// Sort by frequency (most used first)
	for i := 0; i < len(globalColorMatches)-1; i++ {
		for j := i + 1; j < len(globalColorMatches); j++ {
			if globalColorMatches[j].frequency > globalColorMatches[i].frequency {
				globalColorMatches[i], globalColorMatches[j] = globalColorMatches[j], globalColorMatches[i]
			}
		}
	}
	// Create global color palette indices - BLACK AND WHITE ONLY MODE
	globalColors := []int{0, 21} // Black and white only
	colorLimit := 2
	
	// Compute edges for source image if edge detection is enabled
	if g.edgeWeight > 0 {
		g.srcEdges = computeEdges(g.srcPix, g.W*ui.TileSize, g.H*ui.TileSize)
	}
	
	// Process all cells with fixed number of goroutines (greedy approach or warm start)
	var wg sync.WaitGroup
	maxGoroutines := runtime.GOMAXPROCS(0) * 4
	
	// Divide cells among goroutines
	cellsPerWorker := (g.totalCells + maxGoroutines - 1) / maxGoroutines
	
	worker := func(workerID int) {
		defer wg.Done()
		
		startCell := workerID * cellsPerWorker
		// Skip if this worker has no cells to process
		if startCell >= g.totalCells {
			return
		}
		
		endCell := startCell + cellsPerWorker
		if endCell > g.totalCells {
			endCell = g.totalCells
		}
		
		myCells := make([]int, 0, endCell-startCell)
		for c := startCell; c < endCell; c++ {
			myCells = append(myCells, c)
		}
		
		if len(myCells) == 0 {
			return
		}
		
		pixelsPerTile := ui.TileSize * ui.TileSize
		
		// Pre-compute data for all my cells
		type cellData struct {
			cell         int
			base         int
			srcAvgR      int
			srcAvgG      int
			srcAvgB      int
			bestErr      int
			bestT        int
			bestFg       int
			bestBg       int
		}
		
		cells := make([]cellData, len(myCells))
		for i, cell := range myCells {
			base := cell * pixelsPerTile
			
			// Pre-compute source pixel average
			srcAvgR, srcAvgG, srcAvgB := 0, 0, 0
			for j := 0; j < pixelsPerTile; j++ {
				s := g.srcPix[base+j]
				srcAvgR += int(s.R)
				srcAvgG += int(s.G)
				srcAvgB += int(s.B)
			}
			srcAvgR /= pixelsPerTile
			srcAvgG /= pixelsPerTile
			srcAvgB /= pixelsPerTile
			
			// Initialize with black or white (closest match)
			const blackIdx = 0
			const whiteIdx = 21
			
			distToBlack := abs(g.paletteRGB[blackIdx].R-srcAvgR) + abs(g.paletteRGB[blackIdx].G-srcAvgG) + abs(g.paletteRGB[blackIdx].B-srcAvgB)
			distToWhite := abs(g.paletteRGB[whiteIdx].R-srcAvgR) + abs(g.paletteRGB[whiteIdx].G-srcAvgG) + abs(g.paletteRGB[whiteIdx].B-srcAvgB)
			
			bestBg := blackIdx
			if distToWhite < distToBlack {
				bestBg = whiteIdx
			}
			
			// Set initial tile
			g.tilemap.Tiles[cell] = ui.Tile{
				Index:      0,
				Color:      ui.ColorIndex(bestBg),
				Background: ui.ColorIndex(bestBg),
			}
			
			cells[i] = cellData{
				cell:         cell,
				base:         base,
				srcAvgR:      srcAvgR,
				srcAvgG:      srcAvgG,
				srcAvgB:      srcAvgB,
				bestErr:      int(^uint(0) >> 1),
				bestT:        0,
				bestFg:       bestBg,
				bestBg:       bestBg,
			}
		}
		
		// Interleaved search: iterate through all combinations, testing all cells together
		maxIterations := g.maxIter
		if maxIterations == 0 {
			maxIterations = g.tileCount * colorLimit * colorLimit
		}
		
		iterCount := 0
		for _, ti := range g.tileOrder {
			for fgi := 0; fgi < colorLimit; fgi++ {
				fg := globalColors[fgi]
				fgR := g.paletteRGB[fg].R
				fgG := g.paletteRGB[fg].G
				fgB := g.paletteRGB[fg].B
				
				for bgi := 0; bgi < colorLimit; bgi++ {
					bg := globalColors[bgi]
					bgR := g.paletteRGB[bg].R
					bgG := g.paletteRGB[bg].G
					bgB := g.paletteRGB[bg].B
					
					// Skip very similar fg/bg combinations
					if abs(fgR-bgR)+abs(fgG-bgG)+abs(fgB-bgB) < 20 {
						continue
					}
					
					// Test this combination for ALL my cells
					for ci := range cells {
						cd := &cells[ci]
						
						// Quick rejections
						tileAvgDist := abs(g.tileAvgR[ti]-cd.srcAvgR) + abs(g.tileAvgG[ti]-cd.srcAvgG) + abs(g.tileAvgB[ti]-cd.srcAvgB)
						if tileAvgDist > 300 {
							continue
						}
						
						fgAvgDist := abs(fgR-cd.srcAvgR) + abs(fgG-cd.srcAvgG) + abs(fgB-cd.srcAvgB)
						if fgAvgDist > 350 {
							continue
						}
						
						// Calculate error using structural similarity approach
						colorErr := 0
						
						// Pre-compute means for SSIM-like comparison
						srcMeanR, srcMeanG, srcMeanB := 0, 0, 0
						tileMeanR, tileMeanG, tileMeanB := 0, 0, 0
						
						for i := 0; i < pixelsPerTile; i++ {
							s := g.srcPix[cd.base+i]
							srcMeanR += int(s.R)
							srcMeanG += int(s.G)
							srcMeanB += int(s.B)
						}
						srcMeanR /= pixelsPerTile
						srcMeanG /= pixelsPerTile
						srcMeanB /= pixelsPerTile
						
						// Compute tile mean with current colors
						for i := 0; i < pixelsPerTile; i++ {
							fp := g.tilesPix[ti][i]
							fa := int(g.tilesA[ti][i])
							faInv := 255 - fa
							
							fr := int(fp.R) * fgR * fa
							fg_ := int(fp.G) * fgG * fa
							fb := int(fp.B) * fgB * fa
							
							bp := g.tilesPix[g.bgTileIdx][i]
							br := int(bp.R) * bgR * faInv
							bg_ := int(bp.G) * bgG * faInv
							bb := int(bp.B) * bgB * faInv
							
							tileMeanR += (fr + br) / 65025
							tileMeanG += (fg_ + bg_) / 65025
							tileMeanB += (fb + bb) / 65025
						}
						tileMeanR /= pixelsPerTile
						tileMeanG /= pixelsPerTile
						tileMeanB /= pixelsPerTile
						
						// Luminance (mean) difference is most important for overall brightness match
						meanDiffR := srcMeanR - tileMeanR
						meanDiffG := srcMeanG - tileMeanG
						meanDiffB := srcMeanB - tileMeanB
						colorErr = 10 * (3*meanDiffR*meanDiffR + 4*meanDiffG*meanDiffG + 2*meanDiffB*meanDiffB)
						
						if g.edgeWeight > 0 {
							// Edge-weighted error: pixels on edges matter more
							cellX := cd.cell % g.W
							cellY := cd.cell / g.W
							fullWidth := g.W * ui.TileSize
							
							centerX := float32(ui.TileSize) / 2.0
							centerY := float32(ui.TileSize) / 2.0
							sigma := float32(ui.TileSize) / 3.0 // Gaussian spread
							
							for py := 0; py < ui.TileSize; py++ {
								for px := 0; px < ui.TileSize; px++ {
									i := py*ui.TileSize + px
									
									fp := g.tilesPix[ti][i]
									fa := int(g.tilesA[ti][i])
									faInv := 255 - fa
									
									fr := int(fp.R) * fgR * fa
									fg_ := int(fp.G) * fgG * fa
									fb := int(fp.B) * fgB * fa
									
									bp := g.tilesPix[g.bgTileIdx][i]
									br := int(bp.R) * bgR * faInv
									bg_ := int(bp.G) * bgG * faInv
									bb := int(bp.B) * bgB * faInv
									
									or := (fr + br) / 65025
									og := (fg_ + bg_) / 65025
									ob := (fb + bb) / 65025
									
									s := g.srcPix[cd.base+i]
									
									// Structure error (deviation from mean)
									structErrR := (or - tileMeanR) - (int(s.R) - srcMeanR)
									structErrG := (og - tileMeanG) - (int(s.G) - srcMeanG)
									structErrB := (ob - tileMeanB) - (int(s.B) - srcMeanB)
									structErr := 3*structErrR*structErrR + 4*structErrG*structErrG + 2*structErrB*structErrB
									
									// Spatial Gaussian weighting (center pixels matter more)
									dx := float32(px) - centerX
									dy := float32(py) - centerY
									spatialDist := (dx*dx + dy*dy) / (sigma * sigma)
									spatialWeight := 0.5 + 0.5*float32(1.0/(1.0+spatialDist))
									
									// Weight pixel error by edge strength
									srcX := cellX*ui.TileSize + px
									srcY := cellY*ui.TileSize + py
									srcIdx := srcY*fullWidth + srcX
									edgeStrength := g.srcEdges[srcIdx]
									
									// Combined weighting: spatial + edge
									edgeFactor := 1.0 + float32(edgeStrength)/10000.0*g.edgeWeight
									totalWeight := spatialWeight * edgeFactor
									
									colorErr += int(float32(structErr) * totalWeight)
									
									if colorErr > cd.bestErr {
										break
									}
								}
							}
						} else {
							// Standard color-only error
							for i := 0; i < pixelsPerTile; i++ {
								fp := g.tilesPix[ti][i]
								fa := int(g.tilesA[ti][i])
								faInv := 255 - fa
								
								fr := int(fp.R) * fgR * fa
								fg_ := int(fp.G) * fgG * fa
								fb := int(fp.B) * fgB * fa
								
								bp := g.tilesPix[g.bgTileIdx][i]
								br := int(bp.R) * bgR * faInv
								bg_ := int(bp.G) * bgG * faInv
								bb := int(bp.B) * bgB * faInv
								
								or := (fr + br) / 65025
								og := (fg_ + bg_) / 65025
								ob := (fb + bb) / 65025
								
								s := g.srcPix[cd.base+i]
								dr := or - int(s.R)
								dg := og - int(s.G)
								db := ob - int(s.B)
								
								// Perceptual color weighting (green matters most, blue least)
								colorErr += 3*dr*dr + 4*dg*dg + 2*db*db
								
								if colorErr > cd.bestErr {
									break
								}
							}
						}
						
						err := colorErr
						
						if err > cd.bestErr {
							continue
						}
						
						if err < cd.bestErr {
							cd.bestErr = err
							cd.bestT = ti
							cd.bestFg = fg
							cd.bestBg = bg
							
							// Update tilemap immediately
							g.tilemap.Tiles[cd.cell] = ui.Tile{
								Index:      int16(cd.bestT),
								Color:      ui.ColorIndex(cd.bestFg),
								Background: ui.ColorIndex(cd.bestBg),
							}
						}
					}
					
					iterCount++
					if iterCount >= maxIterations {
						goto done
					}
				}
			}
		}
		done:
		
		// Final update for all my cells
		g.mu.Lock()
		for _, cd := range cells {
			g.tilemap.Tiles[cd.cell] = ui.Tile{
				Index:      int16(cd.bestT),
				Color:      ui.ColorIndex(cd.bestFg),
				Background: ui.ColorIndex(cd.bestBg),
			}
			g.completed.Add(1)
		}
		g.mu.Unlock()
	}
	
	// Launch fixed number of workers
	wg.Add(maxGoroutines)
	for i := 0; i < maxGoroutines; i++ {
		go worker(i)
	}
	
	wg.Wait()
	
	// Run evolutionary algorithm with CLIP guidance
	if g.evolutionPopSize > 0 && g.evolutionGens > 0 {
		log.Printf("Starting CLIP-guided evolution...")
		log.Printf("Population=%d, Generations=%d", g.evolutionPopSize, g.evolutionGens)
		
		evolver := NewEvolutionarySearch(g, g.evolutionPopSize, g.evolutionGens)
		solution := evolver.Run()
		
		// Apply evolved solution
		evolver.ApplyToGenerator(solution)
	}
	
	g.done = true
}

// LiveGeneratorScreen wraps LiveGenerator with UI controls
type LiveGeneratorScreen struct {
	W, H      int
	generator *LiveGenerator
	statusText string
}

func NewLiveGeneratorScreen(width, height int, src image.Image) *LiveGeneratorScreen {
	return &LiveGeneratorScreen{
		W:         width,
		H:         height,
		generator: NewLiveGenerator(width, height, src),
	}
}

func (s *LiveGeneratorScreen) Init() ui.Cmd {
	return s.generator.Init()
}

func (s *LiveGeneratorScreen) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	s.generator.Update(msg)
	
	// Update status text
	if s.generator.IsDone() {
		s.statusText = "Generation complete!"
	} else {
		progress := s.generator.Progress()
		s.statusText = "Generating... " + string(rune('0'+int(progress*100)/10)) + string(rune('0'+int(progress*100)%10)) + "%"
	}
	
	return s, nil
}

func (s *LiveGeneratorScreen) View() *ui.TileMap {
	screen := ui.NewTileMap(s.W, s.H, nil)
	
	// Draw the current tilemap state
	tm := s.generator.View()
	return screen.Overlay(tm, 0, 0)
}

func (s *LiveGeneratorScreen) GetGenerator() *LiveGenerator {
	return s.generator
}

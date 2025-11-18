package ui

import (
	"image"
	"image/color"

	"github.com/anthonynsimon/bild/blur"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/transform"
)

type Source struct {
	Tm *Image
	Image image.Image
	errors []int
	currTile int
	N int // Number of colors to search (from palette)
	BlurRadius float64 // Gaussian blur radius for less precise matching
	EdgeWeight float64 // Weight multiplier for edge pixels (0 = no edge detection)
	EmbeddingWeight float64 // Weight for embedding distance (0 = no embedding matching)
	selectedColors []int // Indices of the N best matching colors from palette
	edgeMap image.Image // Edge detection map for weighted matching
	embedder Embedder // Neural network embedder for tile matching (optional interface)
	cellEmbeddings [][]float32 // Pre-computed embeddings for each cell in the tilemap
}

// Embedder interface for computing image embeddings
type Embedder interface {
	ComputeEmbedding(img image.Image) ([]float32, error)
}

func NewSource(tm *Image, image image.Image, n int, blurRadius, edgeWeight, embeddingWeight float64) *Source {
	s := &Source{
		Tm:              tm,
		Image:           image,
		N:               n,
		BlurRadius:      blurRadius,
		EdgeWeight:      edgeWeight,
		EmbeddingWeight: embeddingWeight,
	}
	// Initialize errors slice
	if tm != nil {
		s.errors = make([]int, tm.W*tm.H)
	}
	return s
}

// SetEmbedder sets the embedder for the source
func (s *Source) SetEmbedder(embedder Embedder) {
	s.embedder = embedder
}

func(s *Source) Init() Cmd {
	if s.Image == nil {
		return nil
	}
	if s.Tm == nil {
		w, h := s.Image.Bounds().Dx() / TileSize, s.Image.Bounds().Dy() / TileSize
		s.Tm = NewImage(w, h, nil)
	}
	// Initialize or resize errors slice if needed
	if len(s.errors) != s.Tm.W*s.Tm.H {
		s.errors = make([]int, s.Tm.W*s.Tm.H)
	}
	s.Image = transform.Resize(s.Image, s.Tm.W*TileSize, s.Tm.H*TileSize, transform.Lanczos)
	// Detect edges before blur for better edge preservation
	if s.EdgeWeight > 0 {
		s.edgeMap = effect.EdgeDetection(s.Image, 1.0)
	}
	// Apply gaussian blur for less precise, more forgiving color matching
	if s.BlurRadius > 0 {
		s.Image = blur.Gaussian(s.Image, s.BlurRadius)
	}
	// Select the N best matching colors from the palette
	s.selectBestColors()
	
	// Compute embeddings if embedder is available
	if s.EmbeddingWeight > 0 && s.embedder != nil {
		s.computeEmbeddings()
	}
	
	// Initialize errors with current tile assignments
	for i := range s.Tm.W*s.Tm.H {
		tile := s.Tm.Tiles[i]
		fgc := PaletteRGBA[tile.Color]
		bgc := PaletteRGBA[tile.Background]
		s.errors[i] = s.calculateError(i, int(tile.Index), fgc, bgc)
	}
	s.currTile = 0 // Reset tile counter when reinitializing
	return nil
}

func (s *Source) Update(msg Msg) (Model, Cmd) {
	for range 20 {
		if _, ok := msg.(Tick); ok && s.currTile < tileCount {
			numJobs := s.Tm.W * s.Tm.H
			for i := range numJobs {
				jobs <- workerData{s, i, s.currTile}
			}
			s.currTile++
		}
	}
	return s.Tm.Update(msg)
}

func (s *Source) View() *Image {
	return s.Tm.View()
}

func worker() {
	defer wg.Done()
	for d := range jobs {
		s := d.s
		cell := d.cell
		currTile := d.currTile
		// Use selected colors if available, otherwise use all colors
		colors := s.selectedColors
		if len(colors) == 0 {
			colors = make([]int, len(PaletteRGBA))
			for i := range colors {
				colors[i] = i
			}
		}
		// Loop foreground colors
		for _, fgIdx := range colors {
			fgc := PaletteRGBA[fgIdx]
			// Loop background colors
			for _, bgIdx := range colors {
				err := s.calculateError(cell, currTile, fgc, PaletteRGBA[bgIdx])
				if err < s.errors[cell] {
					s.errors[cell] = err
					s.Tm.Tiles[cell] = Tile{int16(currTile), ColorIndex(fgIdx), ColorIndex(bgIdx)}
				}
			}
		}
	}
}

// computeEmbeddings computes embeddings for each cell in the tilemap
// Note: Tile embeddings are pre-computed globally during asset initialization
func (s *Source) computeEmbeddings() {
	// Compute embeddings for each cell in the tilemap
	s.cellEmbeddings = make([][]float32, s.Tm.W*s.Tm.H)
	successCount := 0
	for cell := range s.Tm.W*s.Tm.H {
		cellX := cell % s.Tm.W
		cellY := cell / s.Tm.W
		
		// Extract the cell region from the source image
		cellImg := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))
		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				srcX := cellX*TileSize + x
				srcY := cellY*TileSize + y
				cellImg.Set(x, y, s.Image.At(srcX, srcY))
			}
		}
		
		embedding, err := s.embedder.ComputeEmbedding(cellImg)
		if err == nil {
			s.cellEmbeddings[cell] = embedding
			successCount++
		}
	}
}

// selectBestColors analyzes the image and selects the N best matching colors from the palette
func (s *Source) selectBestColors() {
	if s.N <= 0 || s.N >= len(PaletteRGBA) {
		// Use all colors if N is invalid
		s.selectedColors = make([]int, len(PaletteRGBA))
		for i := range s.selectedColors {
			s.selectedColors[i] = i
		}
		return
	}

	// Sample pixels from the image (subsample for performance)
	bounds := s.Image.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	// Calculate how many pixels to sample (max 10000 for performance)
	step := 1
	totalPixels := width * height
	if totalPixels > 10000 {
		step = totalPixels / 10000
		if step < 1 {
			step = 1
		}
	}

	// Count how well each palette color matches the image colors
	colorScores := make([]int64, len(PaletteRGBA))
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			c := color.NRGBAModel.Convert(s.Image.At(x, y)).(color.NRGBA)
			
			// Find closest palette color and accumulate inverse distance
			for i, pc := range PaletteRGBA {
				dr := int(c.R) - int(pc.R)
				dg := int(c.G) - int(pc.G)
				db := int(c.B) - int(pc.B)
				distance := dr*dr + dg*dg + db*db
				
				// Use inverse distance as score (closer colors get higher scores)
				// Add 1 to avoid division by zero
				score := 1000000 / (int64(distance) + 1)
				colorScores[i] += score
			}
		}
	}

	// Find the N colors with highest scores
	type colorScore struct {
		index int
		score int64
	}
	
	scores := make([]colorScore, len(PaletteRGBA))
	for i, score := range colorScores {
		scores[i] = colorScore{i, score}
	}
	
	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// Take the top N colors
	s.selectedColors = make([]int, s.N)
	for i := 0; i < s.N && i < len(scores); i++ {
		s.selectedColors[i] = scores[i].index
	}
}

// calculatePixelError computes the pixel-level RGB difference error with edge weighting
func (s *Source) calculatePixelError(cell, ti int, fgc, bgc color.NRGBA) int {
	pixelErr := 0
	// Convert cell index to x, y coordinates in the tilemap
	cellX := cell % s.Tm.W
	cellY := cell / s.Tm.W
	
	// Compare pixels
	for i := range TileSize*TileSize {
		// FG pixel after color scale multiplication
		fp := tilesPix[ti][i]
		fa := float64(fp.A) / 255.0
		fr := float64(fp.R) * float64(fgc.R) / 255.0
		fg_ := float64(fp.G) * float64(fgc.G) / 255.0
		fb_ := float64(fp.B) * float64(fgc.B) / 255.0

		// Alpha composite fg over bg: out = fg + (1-fa)*bg
		or := fa*fr + (1.0-fa)*float64(bgc.R)
		og := fa*fg_ + (1.0-fa)*float64(bgc.G)
		ob := fa*fb_ + (1.0-fa)*float64(bgc.B)

		// Calculate actual pixel position in the image
		pixelX := cellX*TileSize + i % TileSize
		pixelY := cellY*TileSize + i / TileSize
		c := color.NRGBAModel.Convert(s.Image.At(pixelX, pixelY)).(color.NRGBA)
		dr := int(or - float64(c.R))
		dg := int(og - float64(c.G))
		db := int(ob - float64(c.B))
		pErr := dr*dr + dg*dg + db*db
		
		// Weight edge pixels more heavily if edge detection is enabled
		if s.EdgeWeight > 0 && s.edgeMap != nil {
			edgeColor := color.GrayModel.Convert(s.edgeMap.At(pixelX, pixelY)).(color.Gray)
			// Edge strength is in the grayscale value (255 = strong edge, 0 = no edge)
			edgeStrength := float64(edgeColor.Y) / 255.0
			weight := 1.0 + edgeStrength*s.EdgeWeight
			pErr = int(float64(pErr) * weight)
		}
		
		pixelErr += pErr
	}
	
	return pixelErr
}

// calculateEmbeddingDistance computes the L2 distance between tile and cell embeddings
func (s *Source) calculateEmbeddingDistance(cell, ti int) int {
	// Check if embedding matching is enabled and data is available
	if s.EmbeddingWeight <= 0 || s.embedder == nil {
		return 0
	}
	
	// Use global tile embeddings from assets
	if len(tileEmbeddings) <= ti || len(s.cellEmbeddings) <= cell {
		return 0
	}
	
	tileEmb := tileEmbeddings[ti]
	cellEmb := s.cellEmbeddings[cell]
	
	if tileEmb == nil || cellEmb == nil {
		return 0
	}
	
	// Compute L2 distance between embeddings
	embDist := float64(0)
	for i := 0; i < len(tileEmb) && i < len(cellEmb); i++ {
		diff := float64(tileEmb[i] - cellEmb[i])
		embDist += diff * diff
	}
	
	// Scale embedding distance to be comparable to pixel error
	// Embedding distance is typically 0-2 for normalized vectors
	// Scale by a factor to make it comparable (e.g., 100000)
	result := int(embDist * s.EmbeddingWeight * 100000)
	
	return result
}

// calculateError computes the total error combining pixel and embedding distances
func (s *Source) calculateError(cell, ti int, fgc, bgc color.NRGBA) int {
	pixelErr := s.calculatePixelError(cell, ti, fgc, bgc)
	embeddingErr := s.calculateEmbeddingDistance(cell, ti)
	return pixelErr + embeddingErr
}
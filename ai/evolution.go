package ai

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"

	"github.com/SvenDH/go-card-engine/ui"
)

// Individual represents a candidate solution in the population
type Individual struct {
	Genes   []int // Tile indices for each cell [tileIdx, fgColor, bgColor]
	Fitness float64
}

// EvolutionarySearch uses genetic algorithm with CLIP guidance
type EvolutionarySearch struct {
	generator        *LiveGenerator
	populationSize   int
	eliteSize        int
	mutationRate     float64
	crossoverRate    float64
	maxGenerations   int
	tournamentSize   int
	
	population       []Individual
	bestIndividual   *Individual
	generation       int
	
	// Tile cache for semantic-aware mutation
	tileSimilarity   [][]int
}

// NewEvolutionarySearch creates a new evolutionary search optimizer
func NewEvolutionarySearch(gen *LiveGenerator, popSize, maxGen int) *EvolutionarySearch {
	es := &EvolutionarySearch{
		generator:       gen,
		populationSize:  popSize,
		eliteSize:       popSize / 10,
		mutationRate:    0.1,
		crossoverRate:   0.8,
		maxGenerations:  maxGen,
		tournamentSize:  3,
	}
	
	// Try to load tile similarity cache (read-only)
	es.loadTileSimilarityCache()
	
	return es
}

// InitializePopulation creates initial random population (black & white only)
func (es *EvolutionarySearch) InitializePopulation() {
	es.population = make([]Individual, es.populationSize)
	
	cellCount := es.generator.W * es.generator.H
	tileCount := es.generator.tileCount
	
	const blackIdx = 0
	const whiteIdx = 21
	
	log.Printf("Initializing random population of %d individuals (black and white only)", es.populationSize)
	
	for i := 0; i < es.populationSize; i++ {
		genes := make([]int, cellCount*3)
		
		for cell := 0; cell < cellCount; cell++ {
			base := cell * 3
			genes[base] = rand.Intn(tileCount) // Random tile
			
			// 50% chance: black fg / white bg, 50% chance: white fg / black bg
			if rand.Float64() < 0.5 {
				genes[base+1] = blackIdx
				genes[base+2] = whiteIdx
			} else {
				genes[base+1] = whiteIdx
				genes[base+2] = blackIdx
			}
		}
		
		es.population[i] = Individual{
			Genes:   genes,
			Fitness: 0,
		}
	}
}

// EvaluateFitness computes fitness for all individuals in population
func (es *EvolutionarySearch) EvaluateFitness() {
	// Parallel fitness evaluation for speedup
	numWorkers := runtime.GOMAXPROCS(0)
	chunkSize := (len(es.population) + numWorkers - 1) / numWorkers
	
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * chunkSize
			end := start + chunkSize
			if end > len(es.population) {
				end = len(es.population)
			}
			for i := start; i < end; i++ {
				es.population[i].Fitness = es.computeIndividualFitness(&es.population[i])
			}
		}(w)
	}
	wg.Wait()
	
	// Sort by fitness (lower is better)
	sort.Slice(es.population, func(i, j int) bool {
		return es.population[i].Fitness < es.population[j].Fitness
	})
	
	// Track best individual
	if es.bestIndividual == nil || es.population[0].Fitness < es.bestIndividual.Fitness {
		best := es.population[0]
		es.bestIndividual = &Individual{
			Genes:   make([]int, len(best.Genes)),
			Fitness: best.Fitness,
		}
		copy(es.bestIndividual.Genes, best.Genes)
	}
}

// renderIndividual converts an individual's genes into a rendered image
func (es *EvolutionarySearch) renderIndividual(ind *Individual) image.Image {
	g := es.generator
	width := g.W * ui.TileSize
	height := g.H * ui.TileSize
	
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	
	for cell := 0; cell < g.W*g.H; cell++ {
		geneBase := cell * 3
		tileIdx := ind.Genes[geneBase]
		fgColor := ind.Genes[geneBase+1]
		bgColor := ind.Genes[geneBase+2]
		
		cellX := cell % g.W
		cellY := cell / g.W
		
		tilePixels := g.tilesPix[tileIdx]
		fgRGB := ui.PaletteRGBA[fgColor]
		bgRGB := ui.PaletteRGBA[bgColor]
		
		// Render tile to image
		for ty := 0; ty < ui.TileSize; ty++ {
			for tx := 0; tx < ui.TileSize; tx++ {
				tilePx := tilePixels[ty*ui.TileSize+tx]
				
				// Blend with fg/bg colors
				r := (int(tilePx.R)*int(fgRGB.R) + (255-int(tilePx.R))*int(bgRGB.R)) / 255
				g := (int(tilePx.G)*int(fgRGB.G) + (255-int(tilePx.G))*int(bgRGB.G)) / 255
				b := (int(tilePx.B)*int(fgRGB.B) + (255-int(tilePx.B))*int(bgRGB.B)) / 255
				
				px := cellX*ui.TileSize + tx
				py := cellY*ui.TileSize + ty
				img.Set(px, py, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
			}
		}
	}
	
	return img
}

// computeIndividualFitness evaluates a single individual's fitness
func (es *EvolutionarySearch) computeIndividualFitness(ind *Individual) float64 {
	g := es.generator
	totalError := 0.0
	cellCount := g.W * g.H
	pixelsPerTile := ui.TileSize * ui.TileSize
	
	for cell := 0; cell < cellCount; cell++ {
		base := cell * pixelsPerTile
		geneBase := cell * 3
		
		tileIdx := ind.Genes[geneBase]
		fgColor := ind.Genes[geneBase+1]
		bgColor := ind.Genes[geneBase+2]
		
		// Get source and tile pixels
		srcPixels := g.srcPix[base : base+pixelsPerTile]
		tilePixels := g.tilesPix[tileIdx]
		
		// Compute pixel error
		pixelError := 0.0
		for i := 0; i < pixelsPerTile; i++ {
			// Blend tile with fg/bg colors
			tilePx := tilePixels[i]
			blendedR := (int(tilePx.R)*int(ui.PaletteRGBA[fgColor].R) + 
				         (255-int(tilePx.R))*int(ui.PaletteRGBA[bgColor].R)) / 255
			blendedG := (int(tilePx.G)*int(ui.PaletteRGBA[fgColor].G) + 
				         (255-int(tilePx.G))*int(ui.PaletteRGBA[bgColor].G)) / 255
			blendedB := (int(tilePx.B)*int(ui.PaletteRGBA[fgColor].B) + 
				         (255-int(tilePx.B))*int(ui.PaletteRGBA[bgColor].B)) / 255
			
			srcPx := srcPixels[i]
			dr := int(srcPx.R) - blendedR
			dg := int(srcPx.G) - blendedG
			db := int(srcPx.B) - blendedB
			
			// Perceptual color weighting
			pixelError += math.Sqrt(float64(3*dr*dr + 4*dg*dg + 2*db*db))
		}
		
		totalError += pixelError
	}
	
	// Add CLIP guidance penalty if enabled
	if g.clipModel != nil && g.clipEmbedding != nil {
		// Render current individual to an image
		renderedImg := es.renderIndividual(ind)
		
		// Encode with CLIP vision model
		imgEmbedding, err := g.clipModel.EncodeImage(renderedImg)
		if err == nil {
			// Compute similarity to target prompt
			similarity := CLIPSimilarity(g.clipEmbedding, imgEmbedding)
			
			// Convert to penalty (lower similarity = higher penalty)
			// Scale by total pixel error so CLIP has full influence
			clipPenalty := (1.0 - similarity) * float32(totalError)
			totalError += float64(clipPenalty)
		}
	}
	
	return totalError
}

// TournamentSelection selects a parent using tournament selection
func (es *EvolutionarySearch) TournamentSelection() *Individual {
	best := rand.Intn(es.populationSize)
	
	for i := 1; i < es.tournamentSize; i++ {
		contestant := rand.Intn(es.populationSize)
		if es.population[contestant].Fitness < es.population[best].Fitness {
			best = contestant
		}
	}
	
	return &es.population[best]
}

// Crossover performs uniform crossover between two parents
func (es *EvolutionarySearch) Crossover(parent1, parent2 *Individual) Individual {
	child := Individual{
		Genes: make([]int, len(parent1.Genes)),
	}
	
	// Uniform crossover at cell level (keep cell's 3 genes together)
	cellCount := len(parent1.Genes) / 3
	for cell := 0; cell < cellCount; cell++ {
		base := cell * 3
		if rand.Float64() < 0.5 {
			// Take from parent1
			copy(child.Genes[base:base+3], parent1.Genes[base:base+3])
		} else {
			// Take from parent2
			copy(child.Genes[base:base+3], parent2.Genes[base:base+3])
		}
	}
	
	return child
}

// Mutate modifies genes using embedding-aware smooth transitions
func (es *EvolutionarySearch) Mutate(ind *Individual) {
	cellCount := len(ind.Genes) / 3
	tileCount := es.generator.tileCount
	
	for cell := 0; cell < cellCount; cell++ {
		if rand.Float64() < es.mutationRate {
			base := cell * 3
			
			// 90% chance: mutate tile, 10% chance: swap fg/bg
			if rand.Float64() < 0.9 {
				// Mutate tile (black/white only mode)
				if len(es.tileSimilarity) > 0 {
					// 70% chance: pick similar tile, 30% chance: random
					if rand.Float64() < 0.7 {
						currentTile := ind.Genes[base]
						if currentTile < len(es.tileSimilarity) {
							similarTiles := es.tileSimilarity[currentTile]
							if len(similarTiles) > 0 {
								// Pick from top 5 most similar tiles
								n := len(similarTiles)
								if n > 5 {
									n = 5
								}
								ind.Genes[base] = similarTiles[rand.Intn(n)]
							} else {
								ind.Genes[base] = rand.Intn(tileCount)
							}
						} else {
							ind.Genes[base] = rand.Intn(tileCount)
						}
					} else {
						ind.Genes[base] = rand.Intn(tileCount)
					}
				} else {
					ind.Genes[base] = rand.Intn(tileCount)
				}
			} else {
				// Swap foreground and background (reverse black/white)
				fg := ind.Genes[base+1]
				bg := ind.Genes[base+2]
				ind.Genes[base+1] = bg
				ind.Genes[base+2] = fg
			}
		}
	}
}

// loadSimilarityCSV loads a similarity matrix from CSV (read-only)
func loadSimilarityCSV(filename string, expectedSize int) ([][]int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	
	if len(records) != expectedSize {
		return nil, fmt.Errorf("cache size mismatch: expected %d, got %d", expectedSize, len(records))
	}
	
	matrix := make([][]int, len(records))
	for i, record := range records {
		matrix[i] = make([]int, len(record))
		for j, val := range record {
			matrix[i][j], err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid value at [%d][%d]: %w", i, j, err)
			}
		}
	}
	
	return matrix, nil
}

// loadTileSimilarityCache loads pre-computed tile similarity matrix (read-only)
func (es *EvolutionarySearch) loadTileSimilarityCache() {
	// Try to load tile similarity cache if it exists
	cacheFile := ".cache/similarities/tiles.csv"
	
	// Try loading from cache
	if _, err := os.Stat(cacheFile); err == nil {
		tileCount := es.generator.tileCount
		cached, err := loadSimilarityCSV(cacheFile, tileCount)
		if err == nil {
			es.tileSimilarity = cached
			log.Printf("Loaded tile similarity cache (%d tiles)", tileCount)
		}
	}
	// If cache doesn't exist or fails to load, tileSimilarity stays nil
	// Mutation will fall back to random tiles
}

// Evolve runs the evolutionary algorithm for one generation
func (es *EvolutionarySearch) Evolve() {
	es.generation++
	
	// Create new population
	newPopulation := make([]Individual, es.populationSize)
	
	// Elitism: Keep best individuals
	for i := 0; i < es.eliteSize; i++ {
		newPopulation[i] = Individual{
			Genes:   make([]int, len(es.population[i].Genes)),
			Fitness: es.population[i].Fitness,
		}
		copy(newPopulation[i].Genes, es.population[i].Genes)
	}
	
	// Fill rest with offspring
	for i := es.eliteSize; i < es.populationSize; i++ {
		var child Individual
		
		if rand.Float64() < es.crossoverRate {
			// Crossover
			parent1 := es.TournamentSelection()
			parent2 := es.TournamentSelection()
			child = es.Crossover(parent1, parent2)
		} else {
			// Clone from parent
			parent := es.TournamentSelection()
			child = Individual{
				Genes: make([]int, len(parent.Genes)),
			}
			copy(child.Genes, parent.Genes)
		}
		
		// Mutation
		es.Mutate(&child)
		
		newPopulation[i] = child
	}
	
	es.population = newPopulation
}

// Run executes the full evolutionary algorithm
func (es *EvolutionarySearch) Run() *Individual {
	es.InitializePopulation()
	
	log.Printf("Evaluating initial fitness...")
	es.EvaluateFitness()
	
	initialBest := es.population[0].Fitness
	log.Printf("Initial best fitness: %.2f", initialBest)
	
	for gen := 0; gen < es.maxGenerations; gen++ {
		es.Evolve()
		es.EvaluateFitness()
		
		// Show a random individual from the population for live visualization
		// This gives a better sense of the diversity and exploration happening
		randomIdx := rand.Intn(es.populationSize)
		es.ApplyToGenerator(&es.population[randomIdx])
		
		if gen%10 == 0 || gen == es.maxGenerations-1 {
			bestFitness := es.population[0].Fitness
			avgFitness := 0.0
			worstFitness := es.population[len(es.population)-1].Fitness
			
			for _, ind := range es.population {
				avgFitness += ind.Fitness
			}
			avgFitness /= float64(es.populationSize)
			
			improvement := ((initialBest - bestFitness) / initialBest) * 100.0
			
			log.Printf("Gen %d/%d: Best=%.2f (↓%.1f%%), Avg=%.2f, Worst=%.2f", 
				gen+1, es.maxGenerations, bestFitness, improvement, avgFitness, worstFitness)
		}
		
		// Update generator progress
		progress := float32(gen+1) / float32(es.maxGenerations)
		es.generator.completed.Store(int32(progress * float32(es.generator.totalCells)))
	}
	
	finalImprovement := ((initialBest - es.bestIndividual.Fitness) / initialBest) * 100.0
	log.Printf("Evolution complete! Fitness improved by %.1f%%", finalImprovement)
	
	return es.bestIndividual
}

// ApplyToGenerator applies the best solution to the generator's tilemap
func (es *EvolutionarySearch) ApplyToGenerator(solution *Individual) {
	g := es.generator
	cellCount := g.W * g.H
	
	g.mu.Lock()
	defer g.mu.Unlock()
	
	for cell := 0; cell < cellCount; cell++ {
		geneBase := cell * 3
		
		g.tilemap.Tiles[cell].Index = int16(solution.Genes[geneBase])
		g.tilemap.Tiles[cell].Color = ui.ColorIndex(solution.Genes[geneBase+1])
		g.tilemap.Tiles[cell].Background = ui.ColorIndex(solution.Genes[geneBase+2])
	}
	
	// Update completion counter
	g.completed.Store(int32(cellCount))
}


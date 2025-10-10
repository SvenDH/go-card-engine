/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/SvenDH/go-card-engine/ai"
	"github.com/SvenDH/go-card-engine/ui"
)

var (
	precomputeModel  string
	precomputeOutput string
)

// precomputeCmd represents the precompute command
var precomputeCmd = &cobra.Command{
	Use:   "precompute",
	Short: "Precompute tile similarity matrix",
	Long: `Precomputes the similarity between each tile and each reversed color tile 
using embeddings from an ONNX model. Stores the results in a CSV file for faster 
evolutionary algorithm mutations.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default model path if not specified
		if precomputeModel == "" {
			precomputeModel = "models/squeezenet1.1-7.onnx"
		}
		
		// Default output path if not specified
		if precomputeOutput == "" {
			precomputeOutput = ".cache/similarities/tiles.csv"
		}
		
		// Create output directory if it doesn't exist
		outputDir := filepath.Dir(precomputeOutput)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
		
		log.Printf("Loading ONNX model from %s...", precomputeModel)
		embedder, err := ai.NewONNXEmbedder(precomputeModel)
		if err != nil {
			log.Fatalf("Failed to load ONNX model: %v", err)
		}
		defer embedder.Close()
		
		// Get tile dimensions
		tw := ui.TilesSource.Bounds().Dx() / ui.TileSize
		th := ui.TilesSource.Bounds().Dy() / ui.TileSize
		tileCount := tw * th
		
		log.Printf("Found %d tiles (%dx%d)", tileCount, tw, th)
		
		// Extract all tile pixels
		log.Printf("Extracting tile pixels...")
		tilesPix := make([][]color.NRGBA, tileCount)
		for ti := 0; ti < tileCount; ti++ {
			sx := (ti % tw) * ui.TileSize
			sy := (ti / tw) * ui.TileSize
			tilesPix[ti] = make([]color.NRGBA, ui.TileSize*ui.TileSize)
			idx := 0
			for y := 0; y < ui.TileSize; y++ {
				for x := 0; x < ui.TileSize; x++ {
					c := color.NRGBAModel.Convert(ui.TilesSource.At(sx+x, sy+y)).(color.NRGBA)
					tilesPix[ti][idx] = c
					idx++
				}
			}
		}
		
		// Render tiles with black/white colors and compute embeddings
		const blackIdx = 0
		const whiteIdx = 21
		
		log.Printf("Computing embeddings for normal tiles (black fg / white bg)...")
		normalEmbeddings := make([][]float32, tileCount)
		for ti := 0; ti < tileCount; ti++ {
			if ti%10 == 0 {
				log.Printf("  Tile %d/%d", ti, tileCount)
			}
			img := renderTileWithColors(tilesPix[ti], blackIdx, whiteIdx)
			embedding, err := embedder.ComputeEmbedding(img)
			if err != nil {
				log.Fatalf("Failed to compute embedding for tile %d: %v", ti, err)
			}
			normalEmbeddings[ti] = embedding
		}
		
		log.Printf("Computing embeddings for reversed tiles (white fg / black bg)...")
		reversedEmbeddings := make([][]float32, tileCount)
		for ti := 0; ti < tileCount; ti++ {
			if ti%10 == 0 {
				log.Printf("  Tile %d/%d", ti, tileCount)
			}
			img := renderTileWithColors(tilesPix[ti], whiteIdx, blackIdx)
			embedding, err := embedder.ComputeEmbedding(img)
			if err != nil {
				log.Fatalf("Failed to compute embedding for reversed tile %d: %v", ti, err)
			}
			reversedEmbeddings[ti] = embedding
		}
		
		// Compute similarity matrix
		log.Printf("Computing similarity matrix...")
		
		// For each tile (normal), compute similarity to all tiles (normal and reversed)
		// Store indices sorted by similarity (highest first)
		type tileSim struct {
			tileIdx  int
			reversed bool
			similarity float32
		}
		
		similarityRanking := make([][]int, tileCount)
		for ti := 0; ti < tileCount; ti++ {
			if ti%10 == 0 {
				log.Printf("  Tile %d/%d", ti, tileCount)
			}
			
			var similarities []tileSim
			
			// Compare to all normal tiles
			for tj := 0; tj < tileCount; tj++ {
				if ti == tj {
					continue // Skip self
				}
				sim := ai.CosineSimilarity(normalEmbeddings[ti], normalEmbeddings[tj])
				similarities = append(similarities, tileSim{
					tileIdx:    tj,
					reversed:   false,
					similarity: sim,
				})
			}
			
			// Compare to all reversed tiles
			for tj := 0; tj < tileCount; tj++ {
				if ti == tj {
					continue // Skip self (reversed version)
				}
				sim := ai.CosineSimilarity(normalEmbeddings[ti], reversedEmbeddings[tj])
				similarities = append(similarities, tileSim{
					tileIdx:    tj,
					reversed:   true,
					similarity: sim,
				})
			}
			
			// Sort by similarity (descending)
			sort.Slice(similarities, func(i, j int) bool {
				return similarities[i].similarity > similarities[j].similarity
			})
			
			// Store top similar tiles (we'll store all for flexibility)
			similarityRanking[ti] = make([]int, len(similarities))
			for i, sim := range similarities {
				// Store tile index (use negative for reversed tiles)
				if sim.reversed {
					similarityRanking[ti][i] = -sim.tileIdx - 1 // -1 because 0 would be ambiguous
				} else {
					similarityRanking[ti][i] = sim.tileIdx
				}
			}
		}
		
		// Write to CSV
		log.Printf("Writing results to %s...", precomputeOutput)
		file, err := os.Create(precomputeOutput)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		
		writer := csv.NewWriter(file)
		defer writer.Flush()
		
		for ti := 0; ti < tileCount; ti++ {
			row := make([]string, len(similarityRanking[ti]))
			for i, tileIdx := range similarityRanking[ti] {
				row[i] = strconv.Itoa(tileIdx)
			}
			if err := writer.Write(row); err != nil {
				log.Fatalf("Failed to write row %d: %v", ti, err)
			}
		}
		
		log.Printf("Successfully wrote similarity matrix for %d tiles to %s", tileCount, precomputeOutput)
		log.Printf("Each row contains similar tile indices in descending order of similarity")
		log.Printf("Negative indices indicate reversed (white fg / black bg) tiles")
	},
}

// renderTileWithColors renders a tile with specific foreground and background colors
func renderTileWithColors(tilePix []color.NRGBA, fgColorIdx, bgColorIdx int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, ui.TileSize, ui.TileSize))
	
	fgRGB := ui.PaletteRGBA[fgColorIdx]
	bgRGB := ui.PaletteRGBA[bgColorIdx]
	
	for i := 0; i < ui.TileSize*ui.TileSize; i++ {
		tilePx := tilePix[i]
		
		// Blend with fg/bg colors
		r := (int(tilePx.R)*int(fgRGB.R) + (255-int(tilePx.R))*int(bgRGB.R)) / 255
		g := (int(tilePx.G)*int(fgRGB.G) + (255-int(tilePx.G))*int(bgRGB.G)) / 255
		b := (int(tilePx.B)*int(fgRGB.B) + (255-int(tilePx.B))*int(bgRGB.B)) / 255
		
		x := i % ui.TileSize
		y := i / ui.TileSize
		img.Set(x, y, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
	}
	
	return img
}

func init() {
	rootCmd.AddCommand(precomputeCmd)
	
	precomputeCmd.Flags().StringVarP(&precomputeModel, "model", "m", "", "Path to ONNX embedding model (default: models/squeezenet1.1-7.onnx)")
	precomputeCmd.Flags().StringVarP(&precomputeOutput, "out", "o", "", "Output CSV file path (default: .cache/similarities/tiles.csv)")
}

/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/spf13/cobra"

	"github.com/SvenDH/go-card-engine/ai"
	"github.com/SvenDH/go-card-engine/ui"
	"github.com/SvenDH/go-card-engine/ui/screens"
)

var (
	viewSaveOutput      string
	viewWidth           int
	viewHeight          int
	viewColors          int
	viewBlurRadius      float64
	viewEdgeWeight      float64
	viewEmbeddingWeight float64
)

// viewCmd represents the view command
var viewCmd = &cobra.Command{
	Use:   "view [image_file]",
	Short: "View an image as an interactive tilemap with pan and zoom controls",
	Long: `View an image as a tilemap with real-time Source updates.
	
Controls:
  WASD/Arrow Keys - Pan the image
  +/- Keys        - Zoom in/out
  R               - Reset view
  Space           - Save tilemap
  ESC             - Quit

The tilemap updates continuously as you move and zoom the image.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Load the image
		reader, err := os.Open(args[0])
		if err != nil {
			log.Fatalf("Failed to open image: %v", err)
		}
		defer reader.Close()
		
		img, format, err := image.Decode(reader)
		if err != nil {
			log.Fatalf("Failed to decode image: %v", err)
		}
		log.Printf("Loaded %s image: %dx%d", format, img.Bounds().Dx(), img.Bounds().Dy())
		
		// Create the image viewer screen
		screen := screens.NewImageViewer(img, viewWidth, viewHeight, viewColors, viewBlurRadius, viewEdgeWeight, viewEmbeddingWeight, viewSaveOutput)
		
		// Initialize embedder if embedding weight is set
		if viewEmbeddingWeight > 0 {
			log.Printf("Initializing SqueezeNet embedder...")
			// Use final output layer which is guaranteed to exist across model versions
			embedder, err := ai.NewONNXEmbedder("models/squeezenet1.1-7.onnx")
			if err != nil {
				log.Printf("Warning: Failed to initialize embedder: %v", err)
				log.Printf("Continuing without embedding matching...")
			} else {
				defer embedder.Close()
				log.Printf("Computing tile embeddings...")
				if err := ui.SetTileEmbedder(embedder); err != nil {
					log.Printf("Warning: Failed to compute tile embeddings: %v", err)
				} else {
					// Set embedder on screen for cell embeddings
					screen.SetEmbedder(embedder)
					log.Printf("Embedder ready")
				}
			}
		}
		
		// Set up the window
		screenWidth := viewWidth * ui.TileSize
		screenHeight := viewHeight * ui.TileSize
		ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
		ebiten.SetWindowTitle("Image Viewer - " + args[0])
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		
		// Create and run the program
		prog := &ui.Program{
			M:      screen,
			Width:  screenWidth,
			Height: screenHeight,
			//ShowDebug: true,
		}
		
		if err := ebiten.RunGame(prog); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
	
	viewCmd.Flags().StringVarP(&viewSaveOutput, "out", "o", "tilemap.csv", "Output file for saving tilemap")
	viewCmd.Flags().IntVarP(&viewWidth, "width", "W", 32, "Width of the tilemap in tiles")
	viewCmd.Flags().IntVarP(&viewHeight, "height", "H", 32, "Height of the tilemap in tiles")
	viewCmd.Flags().IntVarP(&viewColors, "colors", "c", 16, "Number of palette colors to search (default: 16, max: palette size)")
	viewCmd.Flags().Float64VarP(&viewBlurRadius, "blur", "b", 1.5, "Gaussian blur radius for less precise matching (0 = no blur)")
	viewCmd.Flags().Float64VarP(&viewEdgeWeight, "edge", "e", 2.0, "Edge detection weight multiplier (0 = no edge detection, higher = stronger edge preservation)")
	viewCmd.Flags().Float64VarP(&viewEmbeddingWeight, "embedding", "m", 0.0, "Embedding distance weight (0 = no embedding matching, higher = more semantic matching)")
}

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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/spf13/cobra"

	"github.com/SvenDH/go-card-engine/ui"
	"github.com/SvenDH/go-card-engine/ai"
)

var (
	liveSaveOutput      string
	liveWidth           int
	liveHeight          int
	liveEvolutionPop    int
	liveEvolutionGens   int
	liveCLIPTextModel   string
	liveCLIPVisionModel string
	liveCLIPTokenizer   string
)

// genliveCmd represents the genlive command (CLIP-only)
var genliveCmd = &cobra.Command{
	Use:   "genlive [image_or_tilemap.txt]",
	Short: "CLIP-guided tilemap generation with evolution",
	Long:  `Generate a tilemap using CLIP guidance. Optionally provide an image or tilemap.txt as reference.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var img image.Image
		// No input image - create a blank gray image for CLIP-only mode
		width := liveWidth * 8  // ui.TileSize
		height := liveHeight * 8
		img = image.NewGray(image.Rect(0, 0, width, height))
		log.Printf("No input image - using blank canvas for pure CLIP generation")
		
		// Create live generator screen
		screen := ai.NewLiveGeneratorScreen(liveWidth, liveHeight, img)
		
		screen.GetGenerator().EnableEvolution(liveEvolutionPop, liveEvolutionGens)
		log.Printf("Evolution: population=%d, generations=%d", liveEvolutionPop, liveEvolutionGens)
		
		// Default model paths if not specified
		if liveCLIPTextModel == "" {
			liveCLIPTextModel = "models/clip_text_vit_b32_quantized.onnx"
		}
		if liveCLIPVisionModel == "" {
			liveCLIPVisionModel = "models/clip_vision_vit_b32_quantized.onnx"
		}
		if liveCLIPTokenizer == "" {
			liveCLIPTokenizer = "models/clip_tokenizer.json"
		}
		
		err := screen.GetGenerator().EnableCLIP(
			args[0] + ", 1-bit pixelart, tileset, mosaic",
			liveCLIPTextModel,
			liveCLIPVisionModel,
			liveCLIPTokenizer,
		)
		if err != nil {
			log.Printf("Warning: Failed to enable CLIP: %v", err)
		} else {
			log.Printf("CLIP guidance enabled with prompt: \"%s\"", args[0])
		}
		
		// Run the UI
		screenWidth := liveWidth * ui.TileSize
		screenHeight := liveHeight * ui.TileSize
		ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
		ebiten.SetWindowTitle("Live Tilemap Generation")
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
		
		prog := &ui.Program{
			M:      screen,
			Width:  screenWidth,
			Height: screenHeight,
		}
		if err := ebiten.RunGame(prog); err != nil {
			log.Fatal(err)
		}
		
		// Save output if specified
		if liveSaveOutput != "" && screen.GetGenerator().IsDone() {
			tm := screen.GetGenerator().View()
			if err := tm.Save(liveSaveOutput); err != nil {
				log.Printf("Failed to save output: %v", err)
			} else {
				log.Printf("Saved tilemap to %s", liveSaveOutput)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(genliveCmd)

	genliveCmd.Flags().StringVarP(&liveSaveOutput, "out", "o", "", "Save output tilemap to file when complete")
	genliveCmd.Flags().IntVarP(&liveWidth, "width", "W", 8, "Width of the tilemap")
	genliveCmd.Flags().IntVarP(&liveHeight, "height", "H", 8, "Height of the tilemap")
	genliveCmd.Flags().IntVar(&liveEvolutionPop, "pop-size", 200, "Evolution population size")
	genliveCmd.Flags().IntVar(&liveEvolutionGens, "generations", 500, "Evolution generations")
	genliveCmd.Flags().StringVar(&liveCLIPTextModel, "clip-text-model", "", "Path to CLIP text model (default: models/clip_text_vit_b32_quantized.onnx)")
	genliveCmd.Flags().StringVar(&liveCLIPVisionModel, "clip-vision-model", "", "Path to CLIP vision model (default: models/clip_vision_vit_b32_quantized.onnx)")
	genliveCmd.Flags().StringVar(&liveCLIPTokenizer, "clip-tokenizer", "", "Path to CLIP tokenizer (default: models/clip_tokenizer.json)")
}

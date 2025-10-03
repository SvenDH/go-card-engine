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

	"github.com/spf13/cobra"

	generate "github.com/SvenDH/go-card-engine/generate"
)

var (
	outPath string
	width int
	height int
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new tilemap from an image",
	Long: `Generate a new tilemap from an image`,
	Run: func(cmd *cobra.Command, args []string) {
		reader, err := os.Open(args[0])
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()
		m, _, err := image.Decode(reader)
		if err != nil {
			log.Fatal(err)
		}
		tm := generate.ImageToTileMap(m, width, height)
		err = tm.Save(outPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outPath, "out", "o", "", "Output file for tilemap")
	generateCmd.Flags().IntVarP(&width, "width", "W", 16, "Width of the tilemap")
	generateCmd.Flags().IntVarP(&height, "height", "H", 16, "Height of the tilemap")

}

/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/hajimehoshi/ebiten/v2"
	
	"github.com/SvenDH/go-card-engine/ui"
	"github.com/SvenDH/go-card-engine/ui/screens"
)

const (
	screenWidth  = 600
	screenHeight = 480
)

// playCmd represents the play command
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Window setup
		ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
		ebiten.SetWindowTitle("Tiles (Ebitengine Demo)")

		// Start the game loop
		prog := &ui.Program{
			M: screens.NewCardGame(screenWidth / ui.TileSize, screenHeight / ui.TileSize),
			Width: screenWidth,
			Height: screenHeight,
		}
		if err := ebiten.RunGame(prog); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(playCmd)
}

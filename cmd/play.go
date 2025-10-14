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
	screenWidth  = 1920
	screenHeight = 1080
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
		ebiten.SetWindowSize(screenWidth, screenHeight)
		ebiten.SetWindowTitle("Card game")

		// Start the game loop
		prog := &ui.Program{
			M: screens.NewCardGame(screenWidth / 2 / ui.TileSize, screenHeight / 2 / ui.TileSize),
			Width: screenWidth / 2,
			Height: screenHeight / 2,
		}
		if err := ebiten.RunGame(prog); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(playCmd)
}

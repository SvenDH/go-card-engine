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

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a tilemap file",
	Long: `Edit a tilemap file`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load the tilemap
		tilemap, err := ui.LoadTileMap(args[0])
		if err != nil {
			log.Fatal(err)
		}
		e := screens.NewTileMapEditor(tilemap)

		// Window setup
		ebiten.SetWindowSize(e.W * ui.TileSize * 2, e.H * ui.TileSize * 2)
		ebiten.SetWindowTitle("Editor")

		// Start the game loop
		prog := &ui.Program{M: e, Width: e.W * ui.TileSize, Height: e.H * ui.TileSize}
		if err := ebiten.RunGame(prog); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}

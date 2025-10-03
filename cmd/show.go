/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/hajimehoshi/ebiten/v2"
	
	"github.com/SvenDH/go-card-engine/ui"
)

// TileMapViewer is a simple model that displays a tilemap
type TileMapViewer struct {
	tilemap *ui.TileMap
	width   int
	height  int
}

func NewTileMapViewer(tilemap *ui.TileMap, width, height int) *TileMapViewer {
	return &TileMapViewer{
		tilemap: tilemap,
		width:   width,
		height:  height,
	}
}

func (v *TileMapViewer) Init() ui.Cmd {
	return nil
}

func (v *TileMapViewer) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	switch msg.(type) {
	case ui.KeyEvent:
		// Exit on any key press
		return v, func() ui.Msg { return nil }
	}
	return v, nil
}

func (v *TileMapViewer) View() *ui.TileMap {
	return v.tilemap
}

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show [tilemap-file]",
	Short: "Display a tilemap from a file",
	Long:  `Render and display a tilemap from a tilemap file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Load the tilemap
		tilemap, err := ui.LoadTileMap(args[0])
		if err != nil {
			log.Fatal(err)
		}

		// Calculate window size based on tilemap dimensions
		screenWidth := tilemap.W * ui.TileSize
		screenHeight := tilemap.H * ui.TileSize

		// Window setup
		ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
		ebiten.SetWindowTitle("Viewer")

		// Start the game loop
		prog := &ui.Program{
			M:      NewTileMapViewer(tilemap, screenWidth, screenHeight),
			Width:  screenWidth,
			Height: screenHeight,
		}
		if err := ebiten.RunGame(prog); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}

package ui

import (
	"math"
)

// can be used to aid readability.
type Position float64

func (p Position) value() float64 {
	return math.Min(1, math.Max(0, float64(p)))
}

// Position aliases.
const (
	Top    Position = 0.0
	Bottom Position = 1.0
	Center Position = 0.5
	Left   Position = 0.0
	Right  Position = 1.0
)

func JoinHorizontal(pos Position, tms ...*TileMap) *TileMap {
	if len(tms) == 0 {
		return &TileMap{}
	}
	if len(tms) == 1 {
		return tms[0]
	}
	var (
		// Groups of strings broken into multiple lines
		blocks = make([][]Tiles, len(tms))

		// Max line widths for the above text blocks
		maxHeight int
	)
	// Break text blocks into lines and get max widths for each text block
	for i, tm := range tms {
		blocks[i] = tm.getRows()
		if tm.H > maxHeight {
			maxHeight = tm.H
		}
	}

	// Add extra lines to make each side the same height
	var zones []*Zone
	xOffset := 0
	for i := range blocks {
		n := maxHeight - tms[i].H
		extraLines := make([]Tiles, n)
		for j := range extraLines {
			extraLines[j] = make(Tiles, tms[i].W)
		}
		yOffset := 0
		switch pos { //nolint:exhaustive
		case Top:
			blocks[i] = append(blocks[i], extraLines...)
		case Bottom:
			blocks[i] = append(extraLines, blocks[i]...)
			yOffset = n
		default: // Somewhere in the middle
			split := int(math.Round(float64(n) * pos.value()))
			top := n - split
			bottom := n - top
			blocks[i] = append(extraLines[top:], blocks[i]...)
			blocks[i] = append(blocks[i], extraLines[bottom:]...)
			yOffset = top
		}
		for _, zone := range tms[i].Zones {
			zones = append(zones, zone.Add(xOffset, yOffset))
		}
		xOffset += tms[i].W
	}
	// Merge lines
	var tiles []Tile
	for i := range blocks[0] { // remember, all blocks have the same number of members now
		for _, block := range blocks {
			tiles = append(tiles, block[i]...)
		}
	}
	return &TileMap{
		W: xOffset,
		H: maxHeight,
		Tiles: tiles,
		Zones: zones,
	}
}

func JoinVertical(pos Position, tms ...*TileMap) *TileMap {
	if len(tms) == 0 {
		return &TileMap{}
	}
	if len(tms) == 1 {
		return tms[0]
	}
	var (
		blocks   = make([][]Tiles, len(tms))
		maxWidth int
	)

	for i, tm := range tms {
		blocks[i] = tm.getRows()
		if tm.W > maxWidth {
			maxWidth = tm.W
		}
	}

	var tiles Tiles
	var zones []*Zone
	yOffset := 0
	for i, block := range blocks {
		w := maxWidth - len(block[0])

		switch pos { //nolint:exhaustive
		case Left:
			for _, line := range block {
				tiles = append(tiles, line...)
				tiles = append(tiles, make(Tiles, w)...)
			}
			for _, zone := range tms[i].Zones {
				zones = append(zones, zone.Add(0, yOffset))
			}
		case Right:
			for _, line := range block {
				tiles = append(tiles, make([]Tile, w)...)
				tiles = append(tiles, line...)
			}
			for _, zone := range tms[i].Zones {
				zones = append(zones, zone.Add(w, yOffset))
			}

		default: // Somewhere in the middle
			if w < 1 {
				for _, line := range block {
					tiles = append(tiles, line...)
				}
				break
			}

			split := int(math.Round(float64(w) * pos.value()))
			right := w - split
			left := w - right

			for _, line := range block {
				tiles = append(tiles, make([]Tile, left)...)
				tiles = append(tiles, line...)
				tiles = append(tiles, make([]Tile, right)...)
			}
			for _, zone := range tms[i].Zones {
				zones = append(zones, zone.Add(left, yOffset))
			}
		}
		yOffset += tms[i].H
	}
	return &TileMap{
		W: maxWidth,
		H: yOffset,
		Tiles: tiles,
		Zones: zones,
	}
}
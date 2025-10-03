package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

func (e *TileMap) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for y := range e.H {
		var row []string
		for x := range e.W {
			tile := e.Tiles[x+(y*e.W)]
			row = append(row, fmt.Sprintf("%d", tile.Index))
			row = append(row, fmt.Sprintf("%d", tile.Color))
			row = append(row, fmt.Sprintf("%d", tile.Background))
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func LoadTileMap(path string) (*TileMap, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var tiles Tiles
	var width, height int
	height = len(records)

	for _, record := range records {
		width = len(record) / 3
		for i := range len(record) / 3 {
			index, err := strconv.Atoi(record[i*3])
			if err != nil {
				return nil, err
			}
			color, err := strconv.Atoi(record[i*3+1])
			if err != nil {
				return nil, err
			}
			background, err := strconv.Atoi(record[i*3+2])
			if err != nil {
				return nil, err
			}
			tiles = append(tiles, Tile{int16(index), ColorIndex(color), ColorIndex(background)})
		}
	}

	return &TileMap{W: width, H: height, Tiles: tiles}, nil
}

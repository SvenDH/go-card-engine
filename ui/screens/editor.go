package screens

import (
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SvenDH/go-card-engine/ui"
)

const (
	nCols             = 4
	scrollSensitivity = 50.0
	savePath          = "tilemaps/"
)

type TileButton struct {
	index, color, background int
}

func (tb *TileButton) Init() ui.Cmd {
	return nil
}

func (tb *TileButton) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return tb, nil
}

func (tb *TileButton) View() *ui.TileMap {
	return ui.NewTileMap(1, 1, nil).Set(0, 0, tb.index, tb.color, tb.background)
}

type TileMapEditor struct {
	W, H int
	selectedTile, selectedColor, selectedBackground int
	input *ui.Input
	gridInput ui.Model
	saveButton *ui.Zone
	loadButton *ui.Zone
	fileListOpen bool
	fileList []string
	tileButtons []ui.Model
	paletteButtons []ui.Model
}

func NewTileMapEditor(tilemap *ui.TileMap) *TileMapEditor {
	e := &TileMapEditor{
		W: tilemap.W + nCols,
		H: tilemap.H + 1,
		input: &ui.Input{Text: "new_file", Width: 16, Focussed: true},
		gridInput: &ui.Zone{M: tilemap},
		saveButton: &ui.Zone{M: &ui.Label{Text: "Save", Color: 21, Background: 1}},
		loadButton: &ui.Zone{M: &ui.Label{Text: "Load", Color: 21, Background: 1}},
	}

	e.gridInput.(*ui.Zone).Click = func(msg ui.Msg) ui.Cmd {
		switch m := msg.(type) {
		case ui.MouseEvent:
			if m.Button == ebiten.MouseButtonLeft {
				e.gridInput.(*ui.Zone).M.(*ui.TileMap).Set(
					m.RelX,
					m.RelY,
					e.selectedTile,
					e.selectedColor,
					e.selectedBackground,
				)
			}
		}
		return nil
	}

	e.saveButton.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.Save()
			return nil
		}
		return nil
	}

	e.loadButton.Click = func(msg ui.Msg) ui.Cmd {	
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.fileListOpen = !e.fileListOpen
			if e.fileListOpen {
				// List all .txt files in the save directory
				files, err := os.ReadDir(savePath)
				if err == nil {
					e.fileList = nil
					for _, file := range files {
						if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
							e.fileList = append(e.fileList, file.Name())
						}
					}
				}
			}
		}
		return nil
	}
	nTiles := ui.TilesImage.Bounds().Dx() * ui.TilesImage.Bounds().Dy() / ui.TileSize / ui.TileSize
	for i := range nTiles / 6 {
		e.tileButtons = append(e.tileButtons, &ui.Zone{
			M: &TileButton{index: i, color: 21},
			Click: func(msg ui.Msg) ui.Cmd {
				switch msg.(type) {
				case ui.MouseEvent:
					e.selectedTile = i
				}
				return nil
			},
		})
	}
	for i := range len(ui.Palette) {
		e.paletteButtons = append(e.paletteButtons, &ui.Zone{
			M: &TileButton{index: 55, color: 0, background: i},
			Click: func(msg ui.Msg) ui.Cmd {
				switch m := msg.(type) {
				case ui.MouseEvent:
					if m.Button == ebiten.MouseButtonLeft {
						e.selectedColor = i
					}
					if m.Button == ebiten.MouseButtonRight {
						e.selectedBackground = i
					}
				}
				return nil
			},
		})
	}
	return e
}

func (e *TileMapEditor) Init() ui.Cmd {
	return nil
}

func (e *TileMapEditor) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	e.gridInput, _ = e.gridInput.Update(msg)
	for i := range e.tileButtons {
		e.tileButtons[i], _  = e.tileButtons[i].Update(msg)
	}
	for i := range e.paletteButtons {
		e.paletteButtons[i], _ = e.paletteButtons[i].Update(msg)
	}
	return e, nil
}

func (e *TileMapEditor) View() *ui.TileMap {
	input := e.input.View()
	top := ui.JoinHorizontal(ui.Top, input, e.saveButton.View(), e.loadButton.View())
	var tms []*ui.TileMap
	for i := range len(e.paletteButtons) / nCols {
		var row []*ui.TileMap
		for j := range nCols {
			row = append(row, e.paletteButtons[i*nCols+j].View())
		}
		tms = append(tms, ui.JoinHorizontal(ui.Top, row...))
	}
	for i := range len(e.tileButtons) / nCols {
		var row []*ui.TileMap
		for j := range nCols {
			row = append(row, e.tileButtons[i*nCols+j].View())
		}
		tms = append(tms, ui.JoinHorizontal(ui.Top, row...))
	}
	left := ui.JoinVertical(ui.Left, tms...)
	body := ui.JoinHorizontal(ui.Top, left, e.gridInput.View())
	return ui.JoinVertical(ui.Left, top, body)
}

func (e *TileMapEditor) Save() error {
	return nil
}

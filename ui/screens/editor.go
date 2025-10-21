package screens

import (
	"fmt"
	"os"
	"strconv"
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

func (tb *TileButton) View() *ui.Image {
	return ui.NewImage(1, 1, nil).Set(0, 0, tb.index, tb.color, tb.background)
}

type TileMapEditor struct {
	W, H                                            int
	selectedTile, selectedColor, selectedBackground int
	replaceMode                                     bool
	input                                           *ui.Zone
	widthInput                                      *ui.Zone
	heightInput                                     *ui.Zone
	resizeButton                                    *ui.Zone
	gridInput                                       ui.Model
	saveButton                                      *ui.Zone
	loadButton                                      *ui.Zone
	drawButton                                      *ui.Zone
	replaceButton                                   *ui.Zone
	fileListOpen                                    bool
	fileList                                        []string
	tileButtons                                     []ui.Model
	paletteButtons                                  []ui.Model
}

func NewTileMapEditor(tilemap *ui.Image) *TileMapEditor {
	e := &TileMapEditor{
		W:                  tilemap.W + nCols,
		H:                  tilemap.H + 1,
		selectedTile:       0,
		selectedColor:      21,
		selectedBackground: 1,
		input:              &ui.Zone{M: &ui.Input{Text: "new_file", Width: 10}},
		widthInput:         &ui.Zone{M: &ui.Input{Text: fmt.Sprintf("%d", tilemap.W), Width: 2}},
		heightInput:        &ui.Zone{M: &ui.Input{Text: fmt.Sprintf("%d", tilemap.H), Width: 2}},
		resizeButton:       &ui.Zone{M: ui.Icon("arrow_updown")},
		gridInput:          &ui.Zone{M: tilemap},
		saveButton:         &ui.Zone{M: ui.Icon("file")},
		loadButton:         &ui.Zone{M: ui.Icon("list")},
		drawButton:         &ui.Zone{M: ui.Icon("sword")},
		replaceButton:      &ui.Zone{M: ui.Icon("cycle")},
	}

	// Set up focus groups for input fields
	e.input.SetFocusGroup(e.input, e.widthInput, e.heightInput)
	e.widthInput.SetFocusGroup(e.input, e.widthInput, e.heightInput)
	e.heightInput.SetFocusGroup(e.input, e.widthInput, e.heightInput)

	// Set up directional links for keyboard navigation
	e.input.LinkRight(e.widthInput)
	e.widthInput.LinkLeft(e.input).LinkRight(e.heightInput)
	e.heightInput.LinkLeft(e.widthInput)

	// Set up input field click handlers for focus management
	e.input.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.input.Focus()
		}
		return nil
	}

	e.widthInput.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.widthInput.Focus()
		}
		return nil
	}

	e.heightInput.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.heightInput.Focus()
		}
		return nil
	}

	e.gridInput.(*ui.Zone).Capture = true
	e.gridInput.(*ui.Zone).Click = func(msg ui.Msg) ui.Cmd {
		switch msg.(type) {
		case ui.MouseEvent:
			// Unfocus all input fields when clicking on the grid
			e.input.Unfocus()
			e.widthInput.Unfocus()
			e.heightInput.Unfocus()

			zone := e.gridInput.(*ui.Zone)
			tm := e.gridInput.(*ui.Zone).M.(*ui.Image)
			// Bounds check to prevent crash when clicking outside grid
			if zone.MouseX < 0 || zone.MouseX >= tm.W || zone.MouseY < 0 || zone.MouseY >= tm.H {
				return nil
			}
			if e.replaceMode {
				// In replace mode, replace matching colors with new color
				tileIdx := zone.MouseY*tm.W + zone.MouseX
				if tileIdx >= 0 && tileIdx < len(tm.Tiles) {
					// Replace foreground if it matches selectedBackground
					if int(tm.Tiles[tileIdx].Color) == e.selectedBackground {
						tm.Tiles[tileIdx].Color = ui.ColorIndex(e.selectedColor)
					}
					// Replace background if it matches selectedBackground
					if int(tm.Tiles[tileIdx].Background) == e.selectedBackground {
						tm.Tiles[tileIdx].Background = ui.ColorIndex(e.selectedColor)
					}
				}
			} else {
				tm.Set(
					zone.MouseX,
					zone.MouseY,
					e.selectedTile,
					e.selectedColor,
					e.selectedBackground,
				)
			}
		}
		return nil
	}

	e.gridInput.(*ui.Zone).ContextMenu = func(msg ui.Msg) ui.Cmd {
		// Right-click behavior in replace mode is same as left-click
		// (both replace matching colors)
		return nil
	}

	e.gridInput.(*ui.Zone).Dragged = func(msg ui.Msg) ui.Cmd {
		zone := e.gridInput.(*ui.Zone)
		tm := e.gridInput.(*ui.Zone).M.(*ui.Image)
		// Bounds check to prevent crash when dragging outside grid
		if zone.MouseX < 0 || zone.MouseX >= tm.W || zone.MouseY < 0 || zone.MouseY >= tm.H {
			return nil
		}
		if e.replaceMode {
			// In replace mode while dragging, replace matching colors with new color
			tileIdx := zone.MouseY*tm.W + zone.MouseX
			if tileIdx >= 0 && tileIdx < len(tm.Tiles) {
				// Replace foreground if it matches selectedBackground
				if int(tm.Tiles[tileIdx].Color) == e.selectedBackground {
					tm.Tiles[tileIdx].Color = ui.ColorIndex(e.selectedColor)
				}
				// Replace background if it matches selectedBackground
				if int(tm.Tiles[tileIdx].Background) == e.selectedBackground {
					tm.Tiles[tileIdx].Background = ui.ColorIndex(e.selectedColor)
				}
			}
		} else {
			tm.Set(
				zone.MouseX,
				zone.MouseY,
				e.selectedTile,
				e.selectedColor,
				e.selectedBackground,
			)
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

	e.drawButton.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.replaceMode = false
		}
		return nil
	}

	e.replaceButton.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			e.replaceMode = true
		}
		return nil
	}

	e.resizeButton.Click = func(msg ui.Msg) ui.Cmd {
		if msg.(ui.MouseEvent).Button == ebiten.MouseButtonLeft {
			// Parse width and height from input fields
			newWidth, errW := strconv.Atoi(e.widthInput.M.(*ui.Input).Text)
			newHeight, errH := strconv.Atoi(e.heightInput.M.(*ui.Input).Text)

			if errW == nil && errH == nil && newWidth > 0 && newHeight > 0 {
				// Get current tilemap
				tm := e.gridInput.(*ui.Zone).M.(*ui.Image)

				// Create new tilemap with new size
				newTm := ui.NewImage(newWidth, newHeight, nil)

				// Copy existing tiles to new tilemap (up to min dimensions)
				for y := range min(tm.H, newHeight) {
					for x := range min(tm.W, newWidth) {
						oldIdx := y*tm.W + x
						newIdx := y*newWidth + x
						if oldIdx < len(tm.Tiles) && newIdx < len(newTm.Tiles) {
							newTm.Tiles[newIdx] = tm.Tiles[oldIdx]
						}
					}
				}

				// Replace the tilemap
				e.gridInput.(*ui.Zone).M = newTm
				e.W = newWidth + nCols
				e.H = newHeight + 1
			}
		}
		return nil
	}

	nTiles := ui.TilesImage.Bounds().Dx() * ui.TilesImage.Bounds().Dy() / ui.TileSize / ui.TileSize
	for i := range nTiles / 6 {
		tileIdx := i // Capture value for closure
		e.tileButtons = append(e.tileButtons, &ui.Zone{
			M: &TileButton{index: i, color: 21},
			Click: func(msg ui.Msg) ui.Cmd {
				switch msg.(type) {
				case ui.MouseEvent:
					e.selectedTile = tileIdx
				}
				return nil
			},
		})
	}
	for i := range len(ui.Palette) {
		colorIdx := i // Capture value for closure
		zone := &ui.Zone{
			M: &TileButton{index: 55, color: 0, background: i},
			Click: func(msg ui.Msg) ui.Cmd {
				switch msg.(type) {
				case ui.MouseEvent:
					e.selectedColor = colorIdx
				}
				return nil
			},
			ContextMenu: func(msg ui.Msg) ui.Cmd {
				switch m := msg.(type) {
				case ui.MouseEvent:
					if m.Button == ebiten.MouseButtonRight {
						e.selectedBackground = colorIdx
					}
				}
				return nil
			},
		}
		e.paletteButtons = append(e.paletteButtons, zone)
	}
	return e
}

func (e *TileMapEditor) Init() ui.Cmd {
	e.input.Init()
	e.widthInput.Init()
	e.heightInput.Init()
	e.gridInput.Init()
	for i := range e.tileButtons {
		e.tileButtons[i].Init()
	}
	for i := range e.paletteButtons {
		e.paletteButtons[i].Init()
	}
	return nil
}

func (e *TileMapEditor) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	var m ui.Model
	m, _ = e.input.Update(msg)
	e.input = m.(*ui.Zone)
	m, _ = e.widthInput.Update(msg)
	e.widthInput = m.(*ui.Zone)
	m, _ = e.heightInput.Update(msg)
	e.heightInput = m.(*ui.Zone)
	e.gridInput, _ = e.gridInput.Update(msg)
	for i := range e.tileButtons {
		e.tileButtons[i], _ = e.tileButtons[i].Update(msg)
	}
	for i := range e.paletteButtons {
		e.paletteButtons[i], _ = e.paletteButtons[i].Update(msg)
	}
	return e, nil
}

func (e *TileMapEditor) View() *ui.Image {
	input := e.input.View()
	// Update button appearances based on mode
	if e.replaceMode {
		e.drawButton.M.(*ui.Image).Color("white")
		e.replaceButton.M.(*ui.Image).Color("gray")
	} else {
		e.drawButton.M.(*ui.Image).Color("gray")
		e.replaceButton.M.(*ui.Image).Color("white")
	}
	widthView := e.widthInput.View()
	heightView := e.heightInput.View()
	top := ui.JoinHorizontal(ui.Top, input, e.saveButton.View(), e.loadButton.View(), e.drawButton.View(), e.replaceButton.View(), e.resizeButton.View(), widthView, ui.Text("X"), heightView)
	var tms []*ui.Image
	for i := range len(e.paletteButtons) / nCols {
		var row []*ui.Image
		for j := range nCols {
			colorIdx := i*nCols + j
			// Update palette button color based on selection
			btn := e.paletteButtons[colorIdx].(*ui.Zone).M.(*TileButton)
			if colorIdx == e.selectedColor {
				btn.color = 21 // White for selected foreground color
			} else if colorIdx == e.selectedBackground {
				btn.color = 16 // Green for selected background color
			} else {
				btn.color = 0 // Black for unselected
			}
			row = append(row, e.paletteButtons[colorIdx].View())
		}
		tms = append(tms, ui.JoinHorizontal(ui.Top, row...))
	}
	for i := range len(e.tileButtons) / nCols {
		var row []*ui.Image
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

func (e *TileMapEditor) ReplaceColor(oldColor, newColor int) {
	tm := e.gridInput.(*ui.Zone).M.(*ui.Image)
	for i := range tm.Tiles {
		if int(tm.Tiles[i].Color&ui.ColorMask) == oldColor {
			tm.Tiles[i].Color = ui.ColorIndex(newColor)
		}
	}
}

func (e *TileMapEditor) ReplaceBackground(oldColor, newColor int) {
	tm := e.gridInput.(*ui.Zone).M.(*ui.Image)
	for i := range tm.Tiles {
		if int(tm.Tiles[i].Background&ui.ColorMask) == oldColor {
			tm.Tiles[i].Background = ui.ColorIndex(newColor)
		}
	}
}

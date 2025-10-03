package ui

import "strings"

type Label struct {
	Text     	  string
	Color, Background ColorIndex
}

func (w *Label) Init() Cmd {
	return nil
}

func (w *Label) Update(msg Msg) (Model, Cmd) {
    return w, nil
}

func (w *Label) View() *TileMap {
	width, height := w.calculateSize()
	tm := NewTileMap(width, height, nil)
	tm.Clear(w.Background)
	tm.Color(w.Color)
	tm.SetText(w.Text)
	return tm
}

func (w *Label) calculateSize() (int, int) {
	maxWidth := 0
	lines := strings.Split(w.Text, "\n")
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	return maxWidth, len(lines)
}

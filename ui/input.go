package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	delay    = 30
	interval = 3
)

type Input struct {
	Text     string
	Width	 int
	Focussed bool
	runes    []rune
}

func (w *Input) Init() Cmd {
	return nil
}

// repeatingKeyPressed returns true on first press and then at an interval while held
func repeatingKeyPressed(key ebiten.Key) bool {
    d := inpututil.KeyPressDuration(key)
    if d == 1 {
        return true
    }
    if d >= delay && (d-delay)%interval == 0 {
        return true
    }
    return false
}

func (w *Input) Update(msg Msg) (Model, Cmd) {
    if !w.Focussed {
        return w, nil
    }
	w.runes = ebiten.AppendInputChars(w.runes[:0])
	w.Text += string(w.runes)
	//if repeatingKeyPressed(ebiten.KeyEnter) || repeatingKeyPressed(ebiten.KeyNumpadEnter) {
	//	w.Text += "\n"
	//}
	if repeatingKeyPressed(ebiten.KeyBackspace) {
		if len(w.Text) >= 1 {
			w.Text = w.Text[:len(w.Text)-1]
		}
	}
    // Trim to available rows (Rect.Dy is already in tiles)
    //maxRows := w.Tm.Rect.Dy()
    //parts := strings.Split(w.Text, "\n")
    //if len(parts) > maxRows {
    //    parts = parts[:maxRows]
    //}
    //w.Text = strings.Join(parts, "\n")
    return w, nil
}

func (w *Input) View() *TileMap {
	tm := NewTileMap(w.Width, 1, nil)
	tm.SetText(w.Text)
	return tm
}

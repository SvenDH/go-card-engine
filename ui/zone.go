package ui

import "github.com/hajimehoshi/ebiten/v2"

type Zone struct {
	M            Model
	X, Y, W, H, MouseX, MouseY int
	Capture, hovered bool
	Focussed     bool
	Click        func(msg Msg) Cmd
	ContextMenu  func(msg Msg) Cmd
	Enter        func(msg Msg) Cmd
	Leave        func(msg Msg) Cmd
	Owner        Model
}

func (z *Zone) Add(x, y int) *Zone {
	z.X += x
	z.Y += y
	return z
}

func (z Zone) InBounds(x, y int) bool {
	return x >= z.X && x < z.X+z.W && y >= z.Y && y < z.Y+z.H
}

func (w *Zone) Init() Cmd {
	return nil
}

func (w *Zone) Update(msg Msg) (Model, Cmd) {
	switch m := msg.(type) {
	case MouseEvent:
		// Only handle events for this button's zone
		if m.Zone != nil && m.Zone != w {
			return w, nil
		}

		if w.hovered {
			if w.Click != nil && m.Action == MousePress && m.Button == ebiten.MouseButtonLeft {
				return w, w.Click(m)
			}
			if w.ContextMenu != nil && m.Action == MousePress {
				return w, w.ContextMenu(m)
			}
		}

		// Enter/Leave events are already targeted, no need to check hovered
		if w.Enter != nil && m.Action == MouseEnter {
			return w, w.Enter(m)
		}
		if w.Leave != nil && m.Action == MouseLeave {
			return w, w.Leave(m)
		}
	}
	if !w.Focussed {
		return w, nil
	}
	return w, nil
}

func (z *Zone) View() *TileMap {
	tm := z.M.View()
	z.X = 0
	z.Y = 0
	z.W = tm.W
	z.H = tm.H
	z.Owner = z
	return tm.AddZone(z)
}

package ui

import "github.com/hajimehoshi/ebiten/v2"

type Zone struct {
	M                          Model
	X, Y, W, H, MouseX, MouseY int
	Capture, hovered           bool
	Focussed                   bool
	Draggable                  bool
	Droppable                  bool
	DragData                   interface{}
	Click                      func(msg Msg) Cmd
	ContextMenu                func(msg Msg) Cmd
	Enter                      func(msg Msg) Cmd
	Leave                      func(msg Msg) Cmd
	Moved                      func(msg Msg) Cmd
	Release                    func(msg Msg) Cmd
	Dragged                    func(msg Msg) Cmd
	Drop                       func(msg Msg) Cmd
	Owner                      Model
	// Directional links for navigation
	Up, Down, Left, Right *Zone
	// List of zones to unfocus when this zone gets focus
	focusGroup []*Zone
	// Visual indicators
	dragHovered bool
}

func (z *Zone) Add(x, y int) *Zone {
	z.X += x
	z.Y += y
	return z
}

// Focus sets focus on this zone and unfocuses all zones in its focus group
func (z *Zone) Focus() {
	z.Focussed = true
	for _, other := range z.focusGroup {
		if other != z {
			other.Unfocus()
		}
	}
	// If the underlying model is an Input, set its focus state
	if input, ok := z.M.(*Input); ok {
		input.Focussed = true
	}
}

// Unfocus removes focus from this zone
func (z *Zone) Unfocus() {
	z.Focussed = false
	// If the underlying model is an Input, clear its focus state
	if input, ok := z.M.(*Input); ok {
		input.Focussed = false
	}
}

// SetFocusGroup sets the group of zones that should be unfocused when this zone gets focus
func (z *Zone) SetFocusGroup(zones ...*Zone) {
	z.focusGroup = zones
}

// Link connects this zone to another zone in the specified direction
func (z *Zone) LinkUp(other *Zone) *Zone {
	z.Up = other
	return z
}

func (z *Zone) LinkDown(other *Zone) *Zone {
	z.Down = other
	return z
}

func (z *Zone) LinkLeft(other *Zone) *Zone {
	z.Left = other
	return z
}

func (z *Zone) LinkRight(other *Zone) *Zone {
	z.Right = other
	return z
}

func (z Zone) InBounds(x, y int) bool {
	return x >= z.X && x < z.X+z.W && y >= z.Y && y < z.Y+z.H
}

func (w *Zone) Init() Cmd {
	if w.M != nil {
		return w.M.Init()
	}
	return nil
}

func (w *Zone) Update(msg Msg) (Model, Cmd) {
	switch m := msg.(type) {
	case MouseEvent:
		// For drop events, m.Zone contains the dragged zone (for DragData access)
		// For other events, m.Zone should match this zone
		if m.Action != MouseDragRelease && m.Zone != nil && m.Zone != w {
			return w, nil
		}

		if w.Click != nil && m.Action == MousePress && m.Button == ebiten.MouseButtonLeft {
			return w, w.Click(m)
		}
		if w.ContextMenu != nil && m.Action == MousePress {
			return w, w.ContextMenu(m)
		}
		if w.Release != nil && (m.Action == MouseRelease || m.Action == MouseDragRelease) && m.Button == ebiten.MouseButtonLeft {
			return w, w.Release(m)
		}
		if w.Enter != nil && m.Action == MouseEnter {
			return w, w.Enter(m)
		}
		if w.Leave != nil && m.Action == MouseLeave {
			return w, w.Leave(m)
		}
		if w.Drop != nil && m.Action == MouseDragRelease {
			return w, w.Drop(m)
		}
		if w.Dragged != nil && m.Action == MouseDragMotion {
			return w, w.Dragged(m)
		}
		if w.Moved != nil && m.Action == MouseMotion {
			return w, w.Moved(m)
		}
	case KeyEvent:
		// Handle keyboard navigation between linked zones
		if w.Focussed && m.Pressed {
			switch m.Key {
			case ebiten.KeyArrowUp:
				if w.Up != nil {
					w.Unfocus()
					w.Up.Focus()
					return w, nil
				}
			case ebiten.KeyArrowDown:
				if w.Down != nil {
					w.Unfocus()
					w.Down.Focus()
					return w, nil
				}
			case ebiten.KeyArrowLeft:
				if w.Left != nil {
					w.Unfocus()
					w.Left.Focus()
					return w, nil
				}
			case ebiten.KeyArrowRight:
				if w.Right != nil {
					w.Unfocus()
					w.Right.Focus()
					return w, nil
				}
			case ebiten.KeyTab:
				// Tab moves to the right/down, Shift+Tab moves to the left/up
				if ebiten.IsKeyPressed(ebiten.KeyShift) {
					if w.Left != nil {
						w.Unfocus()
						w.Left.Focus()
					} else if w.Up != nil {
						w.Unfocus()
						w.Up.Focus()
					}
				} else {
					if w.Right != nil {
						w.Unfocus()
						w.Right.Focus()
					} else if w.Down != nil {
						w.Unfocus()
						w.Down.Focus()
					}
				}
				return w, nil
			}
		}
	}
	// Only pass messages to underlying model if focused or if it's not an input
	if w.Focussed || !isInputModel(w.M) {
		if w.M != nil {
			var cmd Cmd
			w.M, cmd = w.M.Update(msg)
			return w, cmd
		}
	}
	return w, nil
}

// Helper function to check if a model is an Input
func isInputModel(m Model) bool {
	_, ok := m.(*Input)
	return ok
}

func (z *Zone) View() *Image {
	tm := z.M.View()
	z.X = 0
	z.Y = 0
	z.W = tm.W
	z.H = tm.H
	z.Owner = z

	// Add visual indicators for draggable/droppable zones
	if z.Draggable && z.hovered {
		// Add subtle highlight for draggable zones
		tm = NewStyle().Border(Borders["thin"]).BorderForeground(Colors["yellow"]).Render(tm)
	}
	if z.Droppable && z.dragHovered {
		// Add highlight for drop zones when dragging over them
		tm = NewStyle().Border(Borders["double"]).BorderForeground(Colors["green"]).Render(tm)
	}

	return tm.AddZone(z)
}

package ui

import "github.com/hajimehoshi/ebiten/v2"

type Zone struct {
	M            Model
	X, Y, W, H, MouseX, MouseY int
	Capture, hovered bool
	Focussed     bool
	Draggable    bool
	Droppable    bool
	DragData     interface{}
	Click        func(msg Msg) Cmd
	ContextMenu  func(msg Msg) Cmd
	Enter        func(msg Msg) Cmd
	Leave        func(msg Msg) Cmd
	Moved		 func(msg Msg) Cmd
	Release		 func(msg Msg) Cmd
	Dragged      func(msg Msg) Cmd
	Drop         func(msg Msg) Cmd
	Owner        Model
	// Visual indicators
	dragHovered  bool
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

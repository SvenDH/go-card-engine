package godot

// handScene keeps track of which cards are in the player's hand and mirrors that
// state into the 3D hand scene.
type handScene struct {
	ui *CardGameUI
}

func newHandScene(ui *CardGameUI) *handScene {
	return &handScene{ui: ui}
}

func (h *handScene) Add(view *cardView) {
	if h == nil || view == nil {
		return
	}
	view.location = "hand"
	if h.ui != nil && h.ui.hand3d != nil {
		h.ui.hand3d.Add(view)
	}
}

func (h *handScene) Remove(view *cardView) {
	if h == nil || view == nil {
		return
	}
	if h.ui != nil && h.ui.hand3d != nil {
		h.ui.hand3d.Detach(view)
	}
}

// Detach removes a card without deleting its mesh (used when moving to the board).
func (h *handScene) Detach(view *cardView) {
	if h == nil || view == nil {
		return
	}
	if h.ui != nil && h.ui.hand3d != nil {
		h.ui.hand3d.Detach(view)
	}
}

func (h *handScene) Layout() {
	if h == nil || h.ui == nil || h.ui.hand3d == nil {
		return
	}
	h.ui.hand3d.Layout()
}

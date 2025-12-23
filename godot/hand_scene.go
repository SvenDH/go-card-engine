package godot

import (
	"math"

	"graphics.gd/classdb/Control"
	"graphics.gd/classdb/Node"
	"graphics.gd/classdb/PropertyTweener"
	"graphics.gd/classdb/Tween"
	"graphics.gd/variant/Float"
	"graphics.gd/variant/Vector2"
)

// handScene arranges and animates the player's hand cards inside a Control node.
type handScene struct {
	ui      *CardGameUI
	area    Control.Instance
	cards   []*cardView
	hovered bool

	cardWidth   float64
	cardHeight  float64
	maxSpacing  float64
	arcHeight   float64
	hoverLift   float64
	handLift    float64
	tweenLength Float.X
}

func newHandScene(ui *CardGameUI, area Control.Instance) *handScene {
	h := &handScene{
		ui:          ui,
		area:        area,
		cards:       []*cardView{},
		cardWidth:   120,
		cardHeight:  170,
		maxSpacing:  140,
		arcHeight:   18,
		hoverLift:   26,
		handLift:    22,
		tweenLength: 0.2,
	}

	area.AsControl().OnMouseEntered(func() {
		h.hovered = true
		h.Layout()
	})
	area.AsControl().OnMouseExited(func() {
		h.hovered = false
		h.Layout()
	})

	return h
}

// Add inserts a card view into the hand and relayouts.
func (h *handScene) Add(view *cardView) {
	if view == nil || view.owner != h.ui.player {
		return
	}
	for _, v := range h.cards {
		if v == view {
			h.Layout()
			return
		}
	}
	if view.button.AsNode().GetParent() == Node.Nil {
		h.area.AsNode().AddChild(view.button.AsNode())
	}
	view.button.AsControl().SetCustomMinimumSize(Vector2.New(h.cardWidth, h.cardHeight))
	h.cards = append(h.cards, view)
	if h.ui.hand3d != nil {
		h.ui.hand3d.Add(view)
	}
	h.Layout()
}

// Remove takes a card view out of the hand and relayouts.
func (h *handScene) Remove(view *cardView) {
	if view == nil {
		return
	}
	for i, v := range h.cards {
		if v == view {
			h.cards = append(h.cards[:i], h.cards[i+1:]...)
			break
		}
	}
	if h.ui.hand3d != nil {
		h.ui.hand3d.Remove(view)
	}
	if h.ui.board3d != nil {
		h.ui.board3d.Remove(view)
	}
	if view.moveTween != Tween.Nil {
		view.moveTween.Kill()
		view.moveTween = Tween.Nil
	}
	if parent := view.button.AsNode().GetParent(); parent != Node.Nil {
		parent.RemoveChild(view.button.AsNode())
	}
	h.Layout()
}

// Detach removes a view from the hand layout but leaves its 3D mesh intact.
func (h *handScene) Detach(view *cardView) {
	if view == nil {
		return
	}
	for i, v := range h.cards {
		if v == view {
			h.cards = append(h.cards[:i], h.cards[i+1:]...)
			break
		}
	}
	h.Layout()
}

// Layout recalculates the arc positions and tweens cards into place.
func (h *handScene) Layout() {
	if len(h.cards) == 0 {
		return
	}

	size := h.area.AsControl().Size()
	minSize := h.area.AsControl().GetMinimumSize()
	width := math.Max(float64(size.X), float64(minSize.X))
	height := math.Max(float64(size.Y), float64(minSize.Y))

	handWidth := width - 20
	if handWidth < h.cardWidth {
		handWidth = h.cardWidth
	}

	spacing := h.maxSpacing
	if len(h.cards) > 1 {
		totalCardWidth := h.cardWidth * float64(len(h.cards))
		if totalCardWidth > handWidth {
			spacing = (handWidth - h.cardWidth) / float64(len(h.cards)-1)
			if spacing < 12 {
				spacing = 12
			}
		} else {
			available := handWidth - h.cardWidth
			if available < 0 {
				available = 0
			}
			calculated := available / float64(len(h.cards)-1)
			if calculated < spacing {
				spacing = calculated
			}
		}
	}

	totalWidth := h.cardWidth + spacing*float64(len(h.cards)-1)
	startX := (handWidth-totalWidth)/2 + 10

	baseY := height - h.cardHeight
	if baseY < 0 {
		baseY = 0
	}
	if h.hovered || h.anyCardHovered() {
		baseY -= h.handLift
		if baseY < 0 {
			baseY = 0
		}
	}

	center := float64(len(h.cards)-1) / 2.0
	for i, view := range h.cards {
		if view == nil {
			continue
		}
		norm := (float64(i) - center) / (center + 0.5)
		arcOffset := math.Pow(norm, 2) * h.arcHeight
		targetX := startX + float64(i)*spacing
		targetY := baseY + arcOffset
		if view.hovered && h.ui.dragging != view {
			targetY -= h.hoverLift
		}
		h.tweenTo(view, Vector2.New(targetX, targetY))
	}
	if h.ui.hand3d != nil {
		h.ui.hand3d.Layout(h.cards)
	}
}

func (h *handScene) tweenTo(view *cardView, target Vector2.XY) {
	if view == nil || h.ui.dragging == view {
		return
	}
	if view.moveTween != Tween.Nil {
		view.moveTween.Kill()
	}
	view.homePos = target

	move := view.button.AsNode().CreateTween()
	view.moveTween = move
	PropertyTweener.Make(move, view.button.AsObject(), "position", target, h.tweenLength).
		SetTrans(Tween.TransQuad).
		SetEase(Tween.EaseInOut)
}

func (h *handScene) anyCardHovered() bool {
	for _, view := range h.cards {
		if view != nil && view.hovered && h.ui.dragging != view {
			return true
		}
	}
	return false
}

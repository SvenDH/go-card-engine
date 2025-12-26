package godot

import (
	"github.com/SvenDH/go-card-engine/engine"

	"graphics.gd/classdb/MeshInstance3D"
)

type cardView struct {
	instance   *engine.CardInstance
	owner      *engine.Player
	location   string
	fieldIndex int
	hovered    bool
	mesh       MeshInstance3D.Instance
}

func (c *CardGameUI) createCardView(card *engine.CardInstance, owner *engine.Player) *cardView {
	view := &cardView{
		instance:   card,
		owner:      owner,
		location:   "deck",
		fieldIndex: -1,
	}
	c.cardViews[card.GetId()] = view
	return view
}

func (c *CardGameUI) removeFromHand(view *cardView) {
	if c.hand != nil {
		c.hand.Remove(view)
	}
}

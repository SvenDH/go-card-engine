package godot

import (
	"github.com/SvenDH/go-card-engine/engine"

	"graphics.gd/classdb/MeshInstance3D"
)

func (c *CardGameUI) onDrawCard(card *engine.CardInstance, owner *engine.Player) {
	view := c.cardViews[card.GetId()]
	if view == nil {
		view = c.createCardView(card, owner)
	}
	view.location = "hand"
	view.fieldIndex = -1
	if owner == c.player {
		if c.hand != nil {
			c.hand.Add(view)
		}
	}
	c.logf("%s drew %s", c.playerName(owner), card.GetName())
}

func (c *CardGameUI) onEnterBoard(card *engine.CardInstance, owner *engine.Player, index int) {
	view := c.cardViews[card.GetId()]
	if view == nil {
		view = c.createCardView(card, owner)
	}
	if index < 0 || c.board3d == nil {
		return
	}
	if index >= c.board3d.SlotCount() {
		return
	}

	c.removeFromHand(view)
	if c.hand3d != nil && view.mesh == MeshInstance3D.Nil {
		c.hand3d.ensureMesh(view)
	}
	if view.mesh != MeshInstance3D.Nil {
		c.board3d.PlaceAt(view, index, owner == c.enemy)
	}
	c.logf("%s placed %s on field %d", c.playerName(owner), card.GetName(), index)
}

func (c *CardGameUI) onLeaveBoard(card *engine.CardInstance) {
	view := c.cardViews[card.GetId()]
	if view == nil || view.fieldIndex < 0 {
		return
	}
	if c.board3d != nil {
		c.board3d.Remove(view)
	}
	view.fieldIndex = -1
}

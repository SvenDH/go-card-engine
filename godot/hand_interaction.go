package godot

import (
	"graphics.gd/classdb/SubViewport"
	"graphics.gd/classdb/SubViewportContainer"
	"graphics.gd/variant/Float"
	"graphics.gd/variant/Vector2"
)

// layoutHand arranges player hand cards in a row with slight spacing.
func (c *CardGameUI) layoutHand() {
	if c.hand != nil {
		c.hand.Layout()
	}
}

func (c *CardGameUI) viewportPosition(global Vector2.XY) Vector2.XY {
	if c.handContainer == SubViewportContainer.Nil || c.handViewport == SubViewport.Nil {
		return global
	}
	containerPos := c.handContainer.AsControl().GlobalPosition()
	containerSize := c.handContainer.AsControl().Size()
	vpSize := c.handViewport.Size()
	if containerSize.X == 0 || containerSize.Y == 0 {
		return global
	}
	x := (global.X - containerPos.X) * Float.X(vpSize.X) / containerSize.X
	y := (global.Y - containerPos.Y) * Float.X(vpSize.Y) / containerSize.Y
	return Vector2.XY{x, y}
}

func (c *CardGameUI) updateHandRaise(global Vector2.XY) {
	if c.hand3d == nil {
		return
	}
	size := c.AsControl().Size()
	if size.Y == 0 {
		return
	}
	threshold := Float.X(0.6)
	if global.Y/Float.X(size.Y) > threshold {
		c.hand3d.SetHandRaise(1)
		return
	}
	c.hand3d.SetHandRaise(0)
}

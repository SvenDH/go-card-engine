package godot

import (
	"fmt"

	"github.com/SvenDH/go-card-engine/engine"

	"graphics.gd/classdb/Label"
)

func (c *CardGameUI) adjustLife(player *engine.Player, delta int) {
	c.logf("%s life %+d", c.playerName(player), delta)
}

func (c *CardGameUI) setPhase(phase string, player *engine.Player) {
	msg := fmt.Sprintf("Phase: %s (%s)", phase, c.playerName(player))
	if c.phaseLabel != Label.Nil {
		c.phaseLabel.SetText(msg)
	}
	c.logf("%s", msg)
}

func (c *CardGameUI) playerName(p *engine.Player) string {
	switch p {
	case c.player:
		return "You"
	case c.enemy:
		return "Enemy"
	default:
		return "Unknown"
	}
}

func (c *CardGameUI) logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
}

func (c *CardGameUI) queue(fn func()) {
	select {
	case c.eventQueue <- fn:
	default:
		go func() { c.eventQueue <- fn }()
	}
}

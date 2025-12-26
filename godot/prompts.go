package godot

import (
	"math/rand"

	"github.com/SvenDH/go-card-engine/engine"
)

func (c *CardGameUI) showPrompt(kind engine.EventType, player *engine.Player, args []any) {
	if player != c.player {
		// Bot answers for the enemy.
		c.botRespond(kind, args)
		return
	}

	c.clearPrompt()

	num := 1
	if len(args) > 0 {
		if v, ok := args[0].(int); ok {
			num = v
		}
	}
	choices := []any{}
	if len(args) > 1 {
		choices = args[1:]
	}

	c.currentPrompt = kind
	c.promptPlayer = player
	c.promptChoices = choices
	c.promptExpected = num
	c.logf("Prompt: %v (%d choices)", kind, len(choices))
}

func (c *CardGameUI) selectCardView(view *cardView) bool {
	if view == nil || c.promptPlayer == nil {
		return false
	}
	for i, choice := range c.promptChoices {
		card, ok := choice.(*engine.CardInstance)
		if !ok || card == nil {
			continue
		}
		if card.GetId() == view.instance.GetId() {
			c.sendSelection(i)
			return true
		}
	}
	return false
}

func (c *CardGameUI) sendSelection(value int) {
	if c.promptPlayer != nil {
		c.promptPlayer.Send(engine.Msg{Selected: []int{value}})
	}
	c.clearPrompt()
}

func (c *CardGameUI) clearPrompt() {
	c.currentPrompt = engine.NoEvent
	c.promptPlayer = nil
	c.promptChoices = nil
	c.promptExpected = 0
}

func (c *CardGameUI) botRespond(kind engine.EventType, args []any) {
	switch kind {
	case engine.EventPromptCard:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		idx := rand.Intn(len(choices))
		c.enemy.Send(engine.Msg{Selected: []int{idx}})
	case engine.EventPromptField:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		c.enemy.Send(engine.Msg{Selected: []int{rand.Intn(len(choices))}})
	case engine.EventPromptAbility:
		c.enemy.Send(engine.Msg{Selected: []int{0}})
	case engine.EventPromptTarget:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		c.enemy.Send(engine.Msg{Selected: []int{rand.Intn(len(choices))}})
	case engine.EventPromptSource:
		// 80% chance to use as source.
		if rand.Float32() < 0.8 {
			c.enemy.Send(engine.Msg{Selected: []int{1}})
		} else {
			c.enemy.Send(engine.Msg{Selected: []int{0}})
		}
	case engine.EventPromptDiscard:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		c.enemy.Send(engine.Msg{Selected: []int{rand.Intn(len(choices))}})
	}
}

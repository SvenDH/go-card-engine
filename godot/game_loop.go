package godot

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/SvenDH/go-card-engine/engine"
)

func (c *CardGameUI) startGameLoop() {
	deck, err := loadDeck()
	if err != nil {
		c.queue(func() {
			msg := fmt.Sprintf("Failed to load cards: %v", err)
			c.logf("%s", msg)
		})
		return
	}

	rand.Seed(time.Now().UnixNano())

	game := engine.NewGame()
	player := game.AddPlayer(deck...)
	enemy := game.AddPlayer(deck...)

	c.game = game
	c.player = player
	c.enemy = enemy
	c.turn = 1

	c.queue(func() {
		c.logf("Game running")
	})

	game.On(engine.AllEvents, c.handleEngineEvent)
	game.Run()

	c.queue(func() {
		c.logf("Game ended")
	})
}

func (c *CardGameUI) handleEngineEvent(event *engine.Event) {
	if c.eventQueue == nil {
		return
	}
	// Copy the event so closures don't race on shared memory.
	ev := *event
	select {
	case c.eventQueue <- func() { c.applyEvent(&ev) }:
	default:
		go func(fn func()) { c.eventQueue <- fn }(func() { c.applyEvent(&ev) })
	}
}

func (c *CardGameUI) applyEvent(event *engine.Event) {
	switch event.Event {
	case engine.EventAtStartPhase:
		c.turn += 1
		c.setPhase("start", event.Player)
	case engine.EventAtDrawPhase:
		c.setPhase("draw", event.Player)
	case engine.EventAtPlayPhase:
		c.setPhase("play", event.Player)
	case engine.EventAtEndPhase:
		c.setPhase("end", event.Player)
	case engine.EventOnDraw:
		if len(event.Args) > 0 {
			if card, ok := event.Args[0].(*engine.CardInstance); ok {
				c.onDrawCard(card, event.Player)
			}
		}
	case engine.EventOnEnterBoard:
		if len(event.Args) >= 2 {
			card, _ := event.Args[0].(*engine.CardInstance)
			if card != nil {
				if idx, ok := event.Args[1].(int); ok {
					c.onEnterBoard(card, event.Player, idx)
				}
			}
		}
	case engine.EventOnLeaveBoard:
		if len(event.Args) > 0 {
			if card, ok := event.Args[0].(*engine.CardInstance); ok {
				c.onLeaveBoard(card)
			}
		}
	case engine.EventOnLoseLife:
		if len(event.Args) > 0 {
			if amount, ok := event.Args[0].(int); ok {
				c.adjustLife(event.Player, -amount)
			}
		}
	case engine.EventOnGainLife:
		if len(event.Args) > 0 {
			if amount, ok := event.Args[0].(int); ok {
				c.adjustLife(event.Player, amount)
			}
		}
	case engine.EventPromptCard:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptField:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptAbility:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptTarget:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptSource:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptDiscard:
		c.showPrompt(event.Event, event.Player, event.Args)
	}
}

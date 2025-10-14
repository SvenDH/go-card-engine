package screens

import (
	"math/rand"
	"time"

	"github.com/SvenDH/go-card-engine/game"
)

// enemyBotPromptCard handles the bot's card selection logic
func (e *CardGame) enemyBotPromptCard(choices []any, player *game.Player) {
	// Add a small delay to make the bot feel more natural
	go func() {
		time.Sleep(300 * time.Millisecond)

		if len(choices) == 0 {
			e.enemy.Send(game.Msg{Selected: []int{game.SkipCode}})
			return
		}

		// 70% chance to play a card, 30% chance to skip
		if rand.Float32() < 0.7 {
			// Select a random card from available choices
			selected := rand.Intn(len(choices))
			// Track the enemy's selected card
			if cardInst, ok := choices[selected].(*game.CardInstance); ok {
				if card, exists := e.cardMap[cardInst.GetId()]; exists {
					e.enemySelectedCard = card
				} else {
					// Create the card if it doesn't exist yet
					e.enemySelectedCard = e.CreateCard(cardInst)
				}
			}
			e.enemy.Send(game.Msg{Selected: []int{selected}})
		} else {
			e.enemySelectedCard = nil
			e.enemy.Send(game.Msg{Selected: []int{game.SkipCode}})
		}
	}()
}

// enemyBotPromptField handles the bot's field selection logic
func (e *CardGame) enemyBotPromptField(choices []any) {
	go func() {
		time.Sleep(200 * time.Millisecond)

		if len(choices) == 0 {
			e.enemy.Send(game.Msg{Selected: []int{game.SkipCode}})
			return
		}

		// Select a random field from available choices
		selected := rand.Intn(len(choices))
		e.enemy.Send(game.Msg{Selected: []int{selected}})
	}()
}

// enemyBotPromptAbility handles the bot's ability selection logic
func (e *CardGame) enemyBotPromptAbility(choices []any) {
	go func() {
		time.Sleep(250 * time.Millisecond)

		if len(choices) == 0 {
			e.enemy.Send(game.Msg{Selected: []int{game.SkipCode}})
			return
		}

		// Always activate first available ability
		e.enemy.Send(game.Msg{Selected: []int{0}})
	}()
}

// enemyBotPromptTarget handles the bot's target selection logic
func (e *CardGame) enemyBotPromptTarget(choices []any) {
	go func() {
		time.Sleep(200 * time.Millisecond)

		if len(choices) == 0 {
			e.enemy.Send(game.Msg{Selected: []int{game.SkipCode}})
			return
		}

		// Select a random target
		selected := rand.Intn(len(choices))
		e.enemy.Send(game.Msg{Selected: []int{selected}})
	}()
}

// enemyBotPromptSource handles the bot's source/spell selection logic
func (e *CardGame) enemyBotPromptSource(choices []any) {
	go func() {
		time.Sleep(150 * time.Millisecond)

		// 80% chance to play as source (option 1), 20% as spell (option 0)
		if rand.Float32() < 0.8 {
			e.enemy.Send(game.Msg{Selected: []int{1}})
		} else {
			e.enemy.Send(game.Msg{Selected: []int{0}})
		}
	}()
}

// enemyBotPromptDiscard handles the bot's discard selection logic
func (e *CardGame) enemyBotPromptDiscard(choices []any) {
	go func() {
		time.Sleep(200 * time.Millisecond)

		if len(choices) == 0 {
			e.enemy.Send(game.Msg{Selected: []int{game.SkipCode}})
			return
		}

		// Discard a random card
		selected := rand.Intn(len(choices))
		e.enemy.Send(game.Msg{Selected: []int{selected}})
	}()
}

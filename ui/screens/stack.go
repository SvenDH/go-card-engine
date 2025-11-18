package screens

import (
	"github.com/SvenDH/go-card-engine/engine"
	"github.com/SvenDH/go-card-engine/ui"
)

// Stack manages the stack zone where cards are placed before entering the board
type Stack struct {
	game       *CardGame
	zone       *ui.Zone
	card       *Card
	fieldIndex int
}

// NewStack creates a new stack component
func NewStack(game *CardGame) *Stack {
	s := &Stack{
		game:       game,
		fieldIndex: -1,
	}

	s.zone = &ui.Zone{
		M: ui.NewImage(12, 16, nil),
		W: 12,
		H: 16,
	}

	return s
}

// SetCard places a card on the stack for the given field index
func (s *Stack) SetCard(card *Card, fieldIndex int) {
	s.card = card
	s.fieldIndex = fieldIndex

	if card != nil {
		card.Disabled = false
		// Position at stack location
		stackX := s.game.W/2 - 6
		stackY := s.game.H/2 - 8
		card.AnimateTo(stackX+1, stackY+1)
	}
}

// Clear removes the card from the stack
func (s *Stack) Clear() {
	s.card = nil
	s.fieldIndex = -1
}

// GetCard returns the card on the stack
func (s *Stack) GetCard() *Card {
	return s.card
}

// GetFieldIndex returns the field index for the stacked card
func (s *Stack) GetFieldIndex() int {
	return s.fieldIndex
}

// HasCard returns whether there's a card on the stack
func (s *Stack) HasCard() bool {
	return s.card != nil
}

// Update updates the stack zone
func (s *Stack) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return s.zone.Update(msg)
}

// View returns the stack zone view
func (s *Stack) View() *ui.Image {
	return s.zone.View()
}

// GetAbilities returns the current abilities on the stack
func (s *Stack) GetAbilities() []*engine.AbilityInstance {
	if s.game == nil || s.game.gameState == nil {
		return nil
	}
	return s.game.gameState.GetStackAbilities()
}

// Zone returns the underlying zone
func (s *Stack) Zone() *ui.Zone {
	return s.zone
}

package screens

import (
	"github.com/SvenDH/go-card-engine/ui"
)

// Hand manages the player's hand zone and card layout
type Hand struct {
	game    *CardGame
	zone    *ui.Zone
	cards   []ui.Model
	hovered bool
}

// NewHand creates a new hand component
func NewHand(game *CardGame, width, height int) *Hand {
	h := &Hand{
		game:  game,
		cards: make([]ui.Model, 0),
	}

	h.zone = &ui.Zone{
		M:         ui.NewTileMap(width-15, 10, nil), // Taller to cover raised cards
		W:         width - 15,
		H:         10,
		Droppable: true,
		Enter: func(msg ui.Msg) ui.Cmd {
			h.hovered = true
			h.Layout()
			return nil
		},
		Leave: func(msg ui.Msg) ui.Cmd {
			h.hovered = false
			h.Layout()
			return nil
		},
		Drop: func(msg ui.Msg) ui.Cmd {
			// Handle dropping cards back to hand (only for non-selected cards)
			event := msg.(ui.MouseEvent)
			if event.Zone != nil && event.Zone.DragData != nil {
				card := event.Zone.DragData.(*Card)
				// Don't allow dropping back to hand if it's the selected card
				if game.selectedCard == card {
					card.AnimateTo(card.originalX, card.originalY)
				} else if game.RemoveCard(card) {
					// Check if card is from board - if so, move it back to hand
					card.Location = CardLocHand
					h.cards = append(h.cards, card)
					h.Layout()
				} else {
					// Can't drop here, animate back to original position
					card.AnimateTo(card.originalX, card.originalY)
				}
			}
			return nil
		},
	}

	return h
}

// AddCard adds a card to the hand
func (h *Hand) AddCard(card *Card) {
	card.Location = CardLocHand
	h.cards = append(h.cards, card)
	h.Layout()
}

// RemoveCard removes a card from the hand
func (h *Hand) RemoveCard(card *Card) bool {
	for i, c := range h.cards {
		if c.(*Card) == card {
			h.cards = append(h.cards[:i], h.cards[i+1:]...)
			h.Layout()
			return true
		}
	}
	return false
}

// Layout positions all cards in the hand with arc effect
func (h *Hand) Layout() {
	if len(h.cards) == 0 {
		return
	}

	// Get first card to know card width
	cardWidth := h.cards[0].(*Card).Picture.W + 2 // +2 for border

	// Calculate spacing based on hand size
	handWidth := h.zone.W - 4 // Margins
	maxSpacing := 11
	totalCardWidth := len(h.cards) * cardWidth

	var spacing int
	if totalCardWidth > handWidth {
		// Cards need to overlap
		spacing = (handWidth - cardWidth) / (len(h.cards) - 1)
		if spacing < 1 {
			spacing = 1 // Minimum spacing to keep cards visible
		}
	} else {
		// Cards can spread out
		spacing = maxSpacing
		if len(h.cards) > 1 {
			availableSpace := handWidth - cardWidth
			calculatedSpacing := availableSpace / (len(h.cards) - 1)
			if calculatedSpacing < spacing {
				spacing = calculatedSpacing
			}
		}
	}

	// Calculate total width of hand
	totalWidth := cardWidth + (len(h.cards)-1)*spacing
	startX := h.zone.X + (handWidth-totalWidth)/2 + 2

	// Check if any card in hand is hovered
	anyCardHovered := false
	for _, c := range h.cards {
		if c.(*Card).hovered || c.(*Card).dragging {
			anyCardHovered = true
			break
		}
	}

	// Position cards with arc effect
	for i, card := range h.cards {
		// X position
		targetX := startX + i*spacing

		// Y position with arc (cards at edges are higher)
		baseY := h.zone.Y - 2 // Raised higher
		if !h.hovered && !anyCardHovered {
			baseY += 8 // Hide hand downwards when not hovered
		}
		// Calculate normalized position from center (-1 to 1)
		center := float64(len(h.cards)-1) / 2.0
		normPos := (float64(i) - center) / (center + 0.5)
		// Arc curve (parabola)
		arcOffset := int(normPos * normPos * 4) // Cards at edges lift up by 4 tiles
		targetY := baseY + arcOffset

		// Animate to new position
		// Always create new animation - this will override any existing tween
		card.(*Card).AnimateTo(targetX, targetY)
	}
}

// Update updates the hand zone
func (h *Hand) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return h.zone.Update(msg)
}

// View returns the hand zone view
func (h *Hand) View() *ui.TileMap {
	return h.zone.View()
}

// Cards returns the cards in hand
func (h *Hand) Cards() []ui.Model {
	return h.cards
}

// IsHovered returns whether the hand is hovered
func (h *Hand) IsHovered() bool {
	return h.hovered
}

// Zone returns the underlying zone
func (h *Hand) Zone() *ui.Zone {
	return h.zone
}

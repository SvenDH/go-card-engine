package screens

import (
	"github.com/SvenDH/go-card-engine/ui"
)

// ResourceZone manages a resource zone (player or enemy)
type ResourceZone struct {
	game     *CardGame
	zone     *ui.Zone
	cards    []ui.Model
	hovered  bool
	isPlayer bool
}

// NewResourceZone creates a new resource zone component
func NewResourceZone(game *CardGame, width, height int, isPlayer bool) *ResourceZone {
	r := &ResourceZone{
		game:     game,
		cards:    make([]ui.Model, 0),
		isPlayer: isPlayer,
	}

	tm := ui.NewTileMap(width-15, 13, nil)

	r.zone = &ui.Zone{
		M:         tm,
		W:         tm.W,
		H:         tm.H,
		Droppable: true,
		Enter: func(msg ui.Msg) ui.Cmd {
			r.hovered = true
			return nil
		},
		Leave: func(msg ui.Msg) ui.Cmd {
			r.hovered = false
			return nil
		},
		Drop: func(msg ui.Msg) ui.Cmd {
			// Only allow drops on player resource zone
			if !r.isPlayer {
				event := msg.(ui.MouseEvent)
				if event.Zone != nil && event.Zone.DragData != nil {
					card := event.Zone.DragData.(*Card)
					card.AnimateTo(card.originalX, card.originalY)
				}
				return nil
			}
			event := msg.(ui.MouseEvent)
			if event.Zone != nil && event.Zone.DragData != nil {
				card := event.Zone.DragData.(*Card)
				game.RemoveCard(card)
				card.Location = CardLocBoard
				r.cards = append(r.cards, card)
				r.Layout()
			}
			return nil
		},
	}

	return r
}

// AddCard adds a card to the resource zone
func (r *ResourceZone) AddCard(card *Card) {
	card.Location = CardLocBoard
	r.cards = append(r.cards, card)
	r.Layout()
}

// RemoveCard removes a card from the resource zone
func (r *ResourceZone) RemoveCard(card *Card) bool {
	for i, c := range r.cards {
		if c.(*Card) == card {
			r.cards = append(r.cards[:i], r.cards[i+1:]...)
			r.Layout()
			return true
		}
	}
	return false
}

// Layout positions all cards in the resource zone
func (r *ResourceZone) Layout() {
	if len(r.cards) == 0 {
		return
	}

	cardWidth := r.cards[0].(*Card).Picture.W + 2 // +2 for border
	zoneWidth := r.zone.W - 4                      // Margins
	spacing := 8                                   // Fixed spacing between resource cards

	// Calculate total width needed
	totalWidth := cardWidth + (len(r.cards)-1)*spacing

	// Center the cards or start from left if too many
	startX := r.zone.X + 2
	if totalWidth < zoneWidth {
		startX = r.zone.X + (zoneWidth-totalWidth)/2 + 2
	} else {
		// Cards need to overlap if too many
		spacing = (zoneWidth - cardWidth) / (len(r.cards) - 1)
		if spacing < 1 {
			spacing = 1
		}
	}

	// Position cards horizontally
	for i, card := range r.cards {
		targetX := startX + i*spacing
		targetY := r.zone.Y
		card.(*Card).AnimateTo(targetX, targetY)
	}
}

// Update updates the resource zone
func (r *ResourceZone) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return r.zone.Update(msg)
}

// View returns the resource zone view
func (r *ResourceZone) View() *ui.TileMap {
	return r.zone.View()
}

// Cards returns the cards in the resource zone
func (r *ResourceZone) Cards() []ui.Model {
	return r.cards
}

// IsHovered returns whether the resource zone is hovered
func (r *ResourceZone) IsHovered() bool {
	return r.hovered
}

// Zone returns the underlying zone
func (r *ResourceZone) Zone() *ui.Zone {
	return r.zone
}

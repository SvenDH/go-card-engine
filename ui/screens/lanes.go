package screens

import (
	"github.com/SvenDH/go-card-engine/game"
	"github.com/SvenDH/go-card-engine/ui"
)

// LaneZone represents a single lane slot with hover state
type LaneZone struct {
	zone    *ui.Zone
	hovered bool
	index   int
}

// Lanes represents a row of lane zones for playing cards
type Lanes struct {
	Game      *CardGame
	zones     []*LaneZone
	cards     []ui.Model
	cardStyle ui.Style
	isPlayer  bool
}

// Init initializes the lanes with interactive zones
func (l *Lanes) Init() ui.Cmd {
	l.zones = make([]*LaneZone, 5)
	for i := range l.zones {
		laneIndex := i // Capture loop variable
		laneZone := &LaneZone{
			index: i,
		}
		laneZone.zone = &ui.Zone{
			M:         ui.NewTileMap(10, 12, nil),
			Droppable: true,
			Enter: func(msg ui.Msg) ui.Cmd {
				laneZone.hovered = true
				return nil
			},
			Leave: func(msg ui.Msg) ui.Cmd {
				laneZone.hovered = false
				return nil
			},
			Drop: func(msg ui.Msg) ui.Cmd {
				// Only allow drops on player lanes
				if !l.isPlayer {
					// Animate back to original position
					event := msg.(ui.MouseEvent)
					if event.Zone != nil && event.Zone.DragData != nil {
						card := event.Zone.DragData.(*Card)
						card.AnimateTo(card.originalX, card.originalY)
					}
					return nil
				}
				// Get dragged card from zone's DragData
				event := msg.(ui.MouseEvent)
				if event.Zone != nil && event.Zone.DragData != nil {
					card := event.Zone.DragData.(*Card)
					if l.Game.CanDropCard(card, laneIndex) {
						// Remove from hand and place on stack
						wasInHand := card.Location == CardLocHand
						l.Game.RemoveCard(card)

						// Put card on the stack zone
						l.Game.stack.SetCard(card, laneIndex)

						// Card is no longer in hand, disable hover but highlight it on stack
						card.hovered = false
						card.hoverOffset = 0

						// Rearrange remaining cards in hand if this card came from hand
						if wasInHand {
							l.Game.LayoutHand()
						}

						// Send field selection to game
						if l.Game.selectedCard == card && l.Game.player != nil {
							l.Game.player.Send(game.Msg{Selected: []int{laneIndex}})
							l.Game.prompting = false
							l.Game.validFields = make(map[int]bool)
							l.Game.selectedCard = nil
						}
					} else {
						// Can't drop here, animate back to original position
						card.AnimateTo(card.originalX, card.originalY)
					}
				}
				return nil
			},
		}
		l.zones[i] = laneZone
	}
	l.cards = make([]ui.Model, 5)
	l.cardStyle = ui.NewStyle().Margin(1)
	return nil
}

// Update handles updates for all lane zones and cards
func (l *Lanes) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	var cmd ui.Cmd
	var cmds []ui.Cmd
	for _, lz := range l.zones {
		_, cmd = lz.zone.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	for _, card := range l.cards {
		if card != nil {
			_, cmd = card.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return l, ui.Batch(cmds...)
}

// View renders the lanes with appropriate highlighting
func (l *Lanes) View() *ui.TileMap {
	// Render all zone backgrounds with hover highlighting
	tms := make([]*ui.TileMap, len(l.zones))
	for i, lz := range l.zones {
		borderColor := ui.Colors["dark-brown"]
		// Highlight valid fields during field prompts (only for player's lanes)
		if l.isPlayer && l.Game != nil && len(l.Game.validFields) > 0 {
			if l.Game.validFields[i] {
				borderColor = ui.Colors["brown"]
			} else {
				borderColor = ui.Colors["dark-brown"]
			}
		} else if lz.hovered {
			borderColor = ui.Colors["brown"]
		}
		zoneStyle := ui.NewStyle().
			Margin(1).
			Border(ui.Borders["roundheavy"]).
			BorderForeground(borderColor)
		tms[i] = zoneStyle.Render(lz.zone.View())
	}
	return ui.JoinHorizontal(ui.Top, tms...)
}

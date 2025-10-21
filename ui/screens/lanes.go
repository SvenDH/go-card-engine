package screens

import (
	"time"

	"github.com/SvenDH/go-card-engine/engine"
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
			M:         ui.NewImage(10, 12, nil),
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
							for i, field := range l.Game.playableFields {
								if field == laneIndex {
									l.Game.player.Send(engine.Msg{Selected: []int{i}})
									l.Game.prompting = false
									l.Game.validFields = make(map[int]bool)
									l.Game.selectedCard = nil
									break
								}
							}
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
func (l *Lanes) View() *ui.Image {
	// Update zone positions based on screen layout
	if l.Game != nil {
		// Calculate lane positions
		laneWidth := 12 // 10 + 2 margin
		totalLanesWidth := laneWidth * 5
		laneStartX := (l.Game.W - totalLanesWidth) / 2

		// Calculate Y position based on whether this is player or enemy lanes
		lanesY := 3
		if l.isPlayer {
			// Player lanes: resources(3) + enemy lanes(14)
			lanesY += 14
		}

		// Set zone coordinates
		for i, lz := range l.zones {
			lz.zone.X = laneStartX + i*laneWidth + 1 // +1 for margin
			lz.zone.Y = lanesY + 1                   // +1 for margin
		}
	}

	// Render all zone backgrounds with hover highlighting
	tms := make([]*ui.Image, len(l.zones))
	for i, lz := range l.zones {
		borderColor := ui.Colors["dark-brown"]
		borderStyle := ui.Borders["roundheavy"]

		// Keyboard focus highlight - HIGHEST PRIORITY
		if l.Game != nil && ((l.isPlayer && l.Game.focusMode == "field" && l.Game.focusFieldIndex == i) ||
			(!l.isPlayer && l.Game.focusMode == "enemy-field" && l.Game.focusFieldIndex == i)) {
			// Keyboard focus highlight - use double border for extra visibility
			borderColor = ui.Colors["white"]
			borderStyle = ui.Borders["roundthick"]
		} else if !l.isPlayer && l.Game != nil && l.Game.attackTargetField == i {
			// Show red indicator for attack target field (for non-player lanes being attacked)
			if l.Game.attackTargetIsPreview {
				// Preview mode - show as long as hovering
				borderColor = ui.Colors["red"]
			} else if !l.Game.attackTargetTime.IsZero() && time.Since(l.Game.attackTargetTime) < 500*time.Millisecond {
				// Post-attack animation - show for 500ms
				borderColor = ui.Colors["red"]
			}
		} else if l.isPlayer && l.Game != nil && len(l.Game.validFields) > 0 {
			// Highlight valid fields during field prompts (only for player's lanes)
			if l.Game.validFields[i] {
				borderColor = ui.Colors["brown"]
			} else {
				borderColor = ui.Colors["dark-brown"]
			}
		} else if lz.hovered {
			borderColor = ui.Colors["brown"]
		}
		zoneStyle := ui.NewStyle().
			Margin(1, 0, 1, 1).
			Border(borderStyle).
			BorderForeground(borderColor)
		tms[i] = zoneStyle.Render(lz.zone.View())
	}
	return ui.JoinHorizontal(ui.Top, tms...)
}

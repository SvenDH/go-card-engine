package screens

import (
	"fmt"

	"github.com/SvenDH/go-card-engine/engine"
	"github.com/SvenDH/go-card-engine/tween"
	"github.com/SvenDH/go-card-engine/ui"
)

const (
	CardLocDeck = iota
	CardLocHand
	CardLocBoard
)

// Card represents a visual card component in the UI
type Card struct {
	X, Y        int
	Name        string
	Picture     *ui.Image
	BorderColor ui.ColorIndex
	BorderStyle ui.Border
	Location    int
	Disabled    bool
	OnClick     ui.Cmd
	OnDrop      ui.Cmd

	input                    *ui.Zone
	dragging                 bool
	dragOffsetX, dragOffsetY int
	originalX, originalY     int
	// Animation
	tweenX    *tween.Tween
	tweenY    *tween.Tween
	sequenceY *tween.Sequence
	animating bool
	// Visual state
	hoverOffset  int
	hovered      bool
	Focused      bool // Keyboard focus indicator
	game         *CardGame
	cardInstance *engine.CardInstance
}

// Init initializes the card component with input handling
func (c *Card) Init() ui.Cmd {
	c.BorderColor = ui.Colors["light-beige"]
	c.BorderStyle = ui.Borders["card"]
	c.input = &ui.Zone{
		M:         c.RenderCard(),
		Draggable: true,
		Capture:   true,
		Enter: func(msg ui.Msg) ui.Cmd {
			if !c.dragging {
				c.hoverOffset = 2
				c.hovered = true
				if c.game != nil && c.Location == CardLocHand {
					c.game.LayoutHand()
				}
			}
			return nil
		},
		Leave: func(msg ui.Msg) ui.Cmd {
			if !c.dragging {
				c.hoverOffset = 0
				c.hovered = false
				if c.game != nil && c.Location == CardLocHand {
					c.game.LayoutHand()
				}
			}
			return nil
		},
		Click: func(msg ui.Msg) ui.Cmd {
			// Handle target selection
			if c.game != nil && c.game.promptingTarget && c.cardInstance != nil {
				// Check if this card is a valid target
				for i, choice := range c.game.targetChoices {
					if cardInst, ok := choice.(*engine.CardInstance); ok {
						if cardInst.GetId() == c.cardInstance.GetId() {
							// Valid target selected - send to game
							c.game.player.Send(engine.Msg{Selected: []int{i}})
							c.game.promptingTarget = false
							c.game.targetChoices = nil
							c.game.targetableCards = nil
							c.game.targetableFields = nil
							return nil
						}
					}
				}
			}

			// Send card selection when dragging starts
			if c.game != nil && c.game.prompting && !c.Disabled && c.cardInstance != nil {
				// Find the index of this card in hand
				for i, card := range c.game.playableCards {
					if card == c {
						c.game.selectedCard = c
						c.game.prompting = false
						// Re-enable all cards
						for _, card := range c.game.hand.Cards() {
							card.(*Card).Disabled = false
						}
						c.game.player.Send(engine.Msg{Selected: []int{i}})
						break
					}
				}
			}
			c.dragging = true
			// Store original position for returning if drop fails
			c.originalX = c.X
			c.originalY = c.Y
			// Use zone's MouseX/MouseY which are properly zone-relative
			c.dragOffsetX = c.input.MouseX
			c.dragOffsetY = c.input.MouseY
			// Set drag data for drop zones to access
			c.input.DragData = c
			return c.OnClick
		},
		Dragged: func(msg ui.Msg) ui.Cmd {
			if c.dragging {
				c.X = msg.(ui.MouseEvent).X/ui.TileSize - c.dragOffsetX
				c.Y = msg.(ui.MouseEvent).Y/ui.TileSize - c.dragOffsetY
			}
			return nil
		},
		Release: func(msg ui.Msg) ui.Cmd {
			if c.dragging {
				c.dragging = false
				c.AnimateTo(c.originalX, c.originalY)
				return c.OnDrop
			}
			return nil
		},
	}
	return nil
}

// Update handles animation and input updates for the card
func (c *Card) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	switch m := msg.(type) {
	case ui.Tick:
		// Update tween animations
		if c.animating && !c.dragging {
			// Handle regular tween animation
			if c.tweenX != nil && c.tweenY != nil {
				xVal, xDone := c.tweenX.Update(m.DeltaTime)
				yVal, yDone := c.tweenY.Update(m.DeltaTime)

				c.X = int(xVal)
				c.Y = int(yVal)

				// Animation is done when both tweens are finished
				if xDone && yDone {
					c.animating = false
					c.tweenX = nil
					c.tweenY = nil
				}
			} else if c.sequenceY != nil {
				// Handle sequence animation (for bump effect)
				yVal, _, seqComplete := c.sequenceY.Update(m.DeltaTime)
				c.Y = int(yVal)

				if seqComplete {
					c.animating = false
					c.sequenceY = nil
				}
			}
		}
	}
	return c.input.Update(msg)
}

// RenderCard creates the visual representation of the card
func (c *Card) RenderCard() *ui.Image {
	// Use different border color for focused cards
	borderColor := c.BorderColor
	if c.Focused {
		borderColor = ui.ColorIndex(ui.BrightMap[c.BorderColor])
	}

	// Use green border for targetable cards
	if c.game != nil && c.game.promptingTarget && c.cardInstance != nil {
		for _, targetCard := range c.game.targetableCards {
			if targetCard == c {
				borderColor = ui.ColorIndex(ui.BrightMap[c.BorderColor])
				break
			}
		}
	}

	borderStyle := ui.NewStyle().
		Border(c.BorderStyle).
		BorderForeground(borderColor).
		BorderBackground(ui.Colors["dark"])

	topStyle := ui.NewStyle().
		Foreground(ui.Colors["dark"]).
		Background(c.BorderColor).
		Width(c.Picture.W)

	pic := c.Picture.View()
	label := ui.Text(c.Name)
	if len(c.Name) > c.Picture.W-2 {
		label = ui.Text(c.Name[:c.Picture.W-2])
	}

	// Build cost icons
	var costIcons *ui.Image
	if c.cardInstance != nil && len(c.cardInstance.Card.Costs) > 0 {
		// Map shorthand color codes to full icon names
		colorToIcon := map[string]string{
			"c": "hearts",
			"s": "spades",
			"o": "diamonds",
			"w": "clubs",
		}
		
		costParts := []*ui.Image{}
		for _, cost := range c.cardInstance.Card.Costs {
			if cost.Color != "" {
				// Convert shorthand color code to full icon name
				iconName := colorToIcon[cost.Color]
				if iconName == "" {
					iconName = cost.Color // Use as-is if not a shorthand
				}
				icon := ui.Icon(iconName)
				icon.Tiles[0].Color = ui.ColorIndex(ui.Colors["dark"])
				icon.Tiles[0].Background = ui.ColorIndex(c.BorderColor)
				costParts = append(costParts, icon)
			} else if cost.Number.Number > 0 || cost.Number.X {
				// Add number text
				numText := ui.Text(cost.Number.String())
				numText.Tiles[0].Color = ui.ColorIndex(ui.Colors["dark"])
				numText.Tiles[0].Background = ui.ColorIndex(c.BorderColor)
				costParts = append(costParts, numText)
			}
		}
		if len(costParts) > 0 {
			costIcons = ui.JoinHorizontal(ui.Top, costParts...)
		}
	}

	top := ui.JoinHorizontal(ui.Top, label.View())
	top = topStyle.Render(top)

	// Join top and picture
	content := ui.JoinVertical(ui.Left, top, pic)

	// Overlay cost icons in the top right corner
	if costIcons != nil {
		costX := c.Picture.W - costIcons.W
		content = content.Overlay(costIcons, costX, 0)
	}

	// Overlay stats at the bottom of the image if available
	if c.cardInstance != nil {
		power := c.cardInstance.GetPower()
		health := c.cardInstance.GetHealth()
		// Check if card has stats (non-zero or X)
		if power.Number != 0 || power.X || health.Number != 0 || health.X {
			statsStr := fmt.Sprintf("%s/%s", power.String(), health.String())

			statsStyle := ui.NewStyle().
				Foreground(ui.Colors["dark"]).
				Background(ui.Colors["light-beige"]).
				Width(c.Picture.W).
				AlignHorizontal(ui.Center)

			statsText := ui.Text(statsStr)
			stats := statsStyle.Render(statsText.View())

			// Overlay stats at the bottom of the content
			statsY := top.H + pic.H - 1
			content = content.Overlay(stats, 0, statsY)
		}
	}

	tm := borderStyle.Render(content)
	if c.Disabled {
		tm.Tiles.Darken(2)
		tm.Tiles.Desaturate()
	}
	return tm
}

// AnimateTo animates the card to a new position
func (c *Card) AnimateTo(x, y int) {
	// Create new tweens for smooth animation
	duration := float32(0.2) // 0.2 seconds duration
	c.tweenX = tween.New(float32(c.X), float32(x), duration, tween.InOutQuad)
	c.tweenY = tween.New(float32(c.Y), float32(y), duration, tween.InOutQuad)
	c.animating = true
}

// AnimateFlyTo animates the card flying to a new position (for discard/destroy effects)
func (c *Card) AnimateFlyTo(x, y int) {
	// Create fast "flying" animation with aggressive easing
	duration := float32(0.3) // Fast flight
	c.tweenX = tween.New(float32(c.X), float32(x), duration, tween.InQuint)
	c.tweenY = tween.New(float32(c.Y), float32(y), duration, tween.InQuint)
	c.animating = true
}

// AnimateBump creates a quick forward-and-back animation
func (c *Card) AnimateBump(direction int) {
	// Bump animation: move forward and back quickly
	// direction: 1 for down (enemy attacking), -1 for up (player attacking)
	forwardDuration := float32(0.08) // Faster forward motion
	backDuration := float32(0.12)    // Slightly slower snap back
	bumpDistance := 14               // Increased distance for more impact

	originalY := c.Y
	targetY := c.Y + (bumpDistance * direction)

	// Create sequence: aggressive forward motion, then snap back
	c.tweenY = nil // Clear regular tween
	c.sequenceY = tween.NewSequence(
		tween.New(float32(originalY), float32(targetY), forwardDuration, tween.OutCubic),
		tween.New(float32(targetY), float32(originalY), backDuration, tween.InBack),
	)
	c.animating = true
}

// View returns the rendered view of the card
func (c *Card) View() *ui.Image {
	c.input.M = c.RenderCard()
	return c.input.View()
}

// GetDrawY returns the Y position adjusted for hover offset
func (c *Card) GetDrawY() int {
	return c.Y - c.hoverOffset
}

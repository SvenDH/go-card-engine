package screens

import (
	"fmt"

	"github.com/SvenDH/go-card-engine/game"
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
	Picture     *ui.TileMap
	BorderColor ui.ColorIndex
	BorderStyle ui.Border
	Location    int
	Highlighted bool
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
	game         *CardGame
	cardInstance *game.CardInstance
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
			// Send card selection when dragging starts
			if c.game != nil && c.game.prompting && c.Highlighted && c.cardInstance != nil {
				// Find the index of this card in hand
				for i, card := range c.game.playableCards {
					if card == c {
						c.game.selectedCard = c
						c.game.player.Send(game.Msg{Selected: []int{i}})
						c.game.prompting = false
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
func (c *Card) RenderCard() *ui.TileMap {
	borderStyle := ui.NewStyle().
		Border(c.BorderStyle).
		BorderForeground(c.BorderColor).
		BorderBackground(ui.Colors["dark"])

	topStyle := ui.NewStyle().
		Foreground(ui.Colors["dark"]).
		Background(c.BorderColor).
		Width(c.Picture.W)

	pic := c.Picture.View()
	label := ui.Text(c.Name)
	top := ui.JoinHorizontal(ui.Top, label.View())
	top = topStyle.Render(top)

	// Add power/health stats at the bottom if available
	var content *ui.TileMap
	if c.cardInstance != nil {
		power := c.cardInstance.GetPower()
		health := c.cardInstance.GetHealth()
		// Check if card has stats (non-zero or X)
		if power.Number != 0 || power.X || health.Number != 0 || health.X {
			statsStr := fmt.Sprintf("%s/%s", power.String(), health.String())

			statsStyle := ui.NewStyle().
				Foreground(ui.Colors["light-beige"]).
				Background(ui.Colors["dark"]).
				Width(c.Picture.W).
				AlignHorizontal(ui.Center)

			statsText := ui.Text(statsStr)
			stats := statsStyle.Render(statsText.View())
			content = ui.JoinVertical(ui.Left, top, pic, stats)
		} else {
			content = ui.JoinVertical(ui.Left, top, pic)
		}
	} else {
		content = ui.JoinVertical(ui.Left, top, pic)
	}

	tm := borderStyle.Render(content)
	if !c.Highlighted {
		tm.Tiles.Darken()
		tm.Tiles.Darken()
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

// AnimateBump creates a quick forward-and-back animation
func (c *Card) AnimateBump(direction int) {
	// Bump animation: move forward and back quickly
	// direction: 1 for down (player attacking), -1 for up (enemy attacking)
	duration := float32(0.15) // Quick bump
	bumpDistance := 8         // Pixels to move

	originalY := c.Y
	targetY := c.Y + (bumpDistance * direction)

	// Create sequence: move to target, then back
	c.tweenY = nil // Clear regular tween
	c.sequenceY = tween.NewSequence(
		tween.New(float32(originalY), float32(targetY), duration, tween.OutQuad),
		tween.New(float32(targetY), float32(originalY), duration, tween.InQuad),
	)
	c.animating = true
}

// View returns the rendered view of the card
func (c *Card) View() *ui.TileMap {
	c.input.M = c.RenderCard()
	return c.input.View()
}

// GetDrawY returns the Y position adjusted for hover offset
func (c *Card) GetDrawY() int {
	return c.Y - c.hoverOffset
}

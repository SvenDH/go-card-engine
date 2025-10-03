package screens

import (
	"github.com/SvenDH/go-card-engine/ui"
)

type CardImage struct {
	Name string
	Picture *ui.TileMap
}

func (c *CardImage) Init() ui.Cmd {
	return nil
}

func (c *CardImage) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return c, nil
}

func (c *CardImage) View() *ui.TileMap {
	borderColor := ui.Colors["light-beige"]
	if c.Name == "DRAW" {
		borderColor = ui.Colors["green"]
	}
	
	borderStyle := ui.NewStyle().
		Border(ui.Borders["card"]).
		BorderForeground(borderColor).
		BorderBackground(ui.Colors["dark"])

	topStyle := ui.NewStyle().
		Foreground(ui.Colors["dark"]).
		Background(borderColor).
		Width(c.Picture.W)

	pic := c.Picture.View()
	label := ui.Text(c.Name)
	top := ui.JoinHorizontal(ui.Top, label.View())
	top = topStyle.Render(top)
	card := ui.JoinVertical(ui.Left, top, pic)
	return borderStyle.Render(card)
}

type Card struct {
	X, Y int
	Name string
	Picture *ui.TileMap
	cardImage *CardImage
	input *ui.Zone
	dragging bool
	dragOffsetX, dragOffsetY int
	originalX, originalY int
	game *CardGame
	// Animation
	animating bool
	targetX, targetY int
	animSpeed float64
	// Visual state
	hoverOffset int
	hovered bool
}

func (c *Card) Init(game *CardGame) ui.Cmd {
	c.game = game
	c.cardImage = &CardImage{
		Name: c.Name,
		Picture: c.Picture,
	}
	c.input = &ui.Zone{
		M: c.cardImage,
		Enter: func(msg ui.Msg) ui.Cmd {
			if !c.dragging && !c.animating {
				c.hoverOffset = 2
				c.hovered = true
			}
			return nil
		},
		Leave: func(msg ui.Msg) ui.Cmd {
			if !c.dragging && !c.animating {
				c.hoverOffset = 0
				c.hovered = false
			}
			return nil
		},
		Click: func(msg ui.Msg) ui.Cmd {
			c.dragging = true
			// Store original position for returning if drop fails
			c.originalX = c.X
			c.originalY = c.Y
			// Use zone's MouseX/MouseY which are properly zone-relative
			c.dragOffsetX = c.input.MouseX
			c.dragOffsetY = c.input.MouseY
			return nil
		},
		Capture: true,
	}
	return nil
}

func (c *Card) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	switch m := msg.(type) {
	case ui.MouseEvent:
		if c.dragging {
			if m.Action == ui.MouseMotion {
				// Update card position during drag
				c.X = m.X/ui.TileSize - c.dragOffsetX
				c.Y = m.Y/ui.TileSize - c.dragOffsetY
			} else if m.Action == ui.MouseRelease {
				// Stop dragging on mouse release
				c.dragging = false
				// Check if card can be dropped in a zone
				if c.game != nil {
					c.game.TryDropCard(c, m.X/ui.TileSize, m.Y/ui.TileSize)
					// Re-layout hand after drop
					c.game.LayoutHand()
				}
			}
		}
	case ui.Tick:
		// Smooth animation
		if c.animating && !c.dragging {
			dx := c.targetX - c.X
			dy := c.targetY - c.Y
			distSq := dx*dx + dy*dy
			
			// If very close or movement would be 0, snap to target
			if distSq <= 4 {
				c.X = c.targetX
				c.Y = c.targetY
				c.animating = false
			} else {
				// Lerp towards target with minimum 1 tile movement
				moveX := int(float64(dx) * c.animSpeed)
				moveY := int(float64(dy) * c.animSpeed)
				
				// Ensure at least 1 tile movement if not at target
				if moveX == 0 && dx != 0 {
					if dx > 0 {
						moveX = 1
					} else {
						moveX = -1
					}
				}
				if moveY == 0 && dy != 0 {
					if dy > 0 {
						moveY = 1
					} else {
						moveY = -1
					}
				}
				
				c.X += moveX
				c.Y += moveY
			}
		}
	}
	return c.input.Update(msg)
}

func (c *Card) AnimateTo(x, y int) {
	c.targetX = x
	c.targetY = y
	c.animating = true
	if c.animSpeed == 0 {
		c.animSpeed = 0.3 // Default animation speed
	}
}

func (c *Card) View() *ui.TileMap {
	return c.input.View()
}

func (c *Card) GetDrawY() int {
	return c.Y - c.hoverOffset
}

type DropZone struct {
	X, Y, W, H int
	Name string
}

func (z *DropZone) Contains(x, y int) bool {
	return x >= z.X && x < z.X+z.W && y >= z.Y && y < z.Y+z.H
}

type CardGame struct {
	W, H int
	deck []*Card
	hand []*Card
	board []*Card
	boardZones []*DropZone
	handZone *DropZone
	drawButton *ui.Zone
}

func NewCardGame(width, height int) *CardGame {
	e := &CardGame{
		W: width,
		H: height,
		deck: make([]*Card, 0),
		hand: make([]*Card, 0),
		board: make([]*Card, 0),
		boardZones: make([]*DropZone, 5),
	}
	
	// Create 5 board zones with better spacing
	for i := range 5 {
		e.boardZones[i] = &DropZone{
			X: 2 + i*13,
			Y: 3,
			W: 11,
			H: 10,
			Name: "Board",
		}
	}
	
	// Create hand zone at bottom
	e.handZone = &DropZone{
		X: 0,
		Y: height - 12,
		W: width - 15,
		H: 12,
		Name: "Hand",
	}
	
	return e
}

func (e *CardGame) Init() ui.Cmd {
	img, _ := ui.LoadTileMap("frog.txt")
	
	// Create deck with 15 cards
	for range 15 {
		card := &Card{
			X: e.W - 13,
			Y: 3,
			Name: "Frog",
			Picture: img,
			animSpeed: 0.25,
		}
		card.Init(e)
		e.deck = append(e.deck, card)
	}
	
	// Create draw button
	buttonText := ui.Text("DRAW")
	buttonModel := &CardImage{
		Name: "DRAW",
		Picture: buttonText,
	}
	buttonModel.Init()
	e.drawButton = &ui.Zone{
		M: buttonModel,
		Click: func(msg ui.Msg) ui.Cmd {
			e.DrawCard()
			return nil
		},
		Capture: true,
	}
	
	return nil
}

func (e *CardGame) DrawCard() {
	if len(e.deck) > 0 {
		// Take card from top of deck
		card := e.deck[len(e.deck)-1]
		e.deck = e.deck[:len(e.deck)-1]
		
		// Add to hand first
		e.hand = append(e.hand, card)
		
		// Recalculate all card positions with new hand size
		e.LayoutHand()
	}
}

func (e *CardGame) LayoutHand() {
	if len(e.hand) == 0 {
		return
	}
	
	// Get first card to know card width
	cardWidth := e.hand[0].Picture.W + 2 // +2 for border
	
	// Calculate spacing based on hand size
	handWidth := e.handZone.W - 4 // Margins
	maxSpacing := 11
	totalCardWidth := len(e.hand) * cardWidth
	
	var spacing int
	if totalCardWidth > handWidth {
		// Cards need to overlap
		spacing = (handWidth - cardWidth) / (len(e.hand) - 1)
		if spacing < 1 {
			spacing = 1 // Minimum spacing to keep cards visible
		}
	} else {
		// Cards can spread out
		spacing = maxSpacing
		if len(e.hand) > 1 {
			availableSpace := handWidth - cardWidth
			calculatedSpacing := availableSpace / (len(e.hand) - 1)
			if calculatedSpacing < spacing {
				spacing = calculatedSpacing
			}
		}
	}
	
	// Calculate total width of hand
	totalWidth := cardWidth + (len(e.hand)-1)*spacing
	startX := e.handZone.X + (handWidth-totalWidth)/2 + 2
	
	// Position cards with arc effect
	for i, card := range e.hand {
		// X position
		targetX := startX + i*spacing
		
		// Y position with arc (cards at edges are higher)
		baseY := e.handZone.Y - 2 // Raised higher
		// Calculate normalized position from center (-1 to 1)
		center := float64(len(e.hand)-1) / 2.0
		normPos := (float64(i) - center) / (center + 0.5)
		// Arc curve (parabola)
		arcOffset := int(normPos * normPos * 4) // Cards at edges lift up by 4 tiles
		targetY := baseY + arcOffset
		
		// Animate to new position
		if card.animating {
			// Already animating (just drawn), update target
			card.targetX = targetX
			card.targetY = targetY
		} else {
			card.AnimateTo(targetX, targetY)
		}
	}
}

func (e *CardGame) CanDropCard(card *Card, zone *DropZone) bool {
	// For now, always return true
	return true
}

func (e *CardGame) TryDropCard(card *Card, x, y int) {
	// Check board zones
	for _, zone := range e.boardZones {
		if zone.Contains(x, y) {
			if e.CanDropCard(card, zone) {
				// Remove from hand and add to board
				e.RemoveCardFromHand(card)
				e.board = append(e.board, card)
				// Animate to zone center
				targetX := zone.X + zone.W/2 - card.Picture.W/2
				targetY := zone.Y + zone.H/2 - card.Picture.H/2
				card.AnimateTo(targetX, targetY)
				// Card is no longer in hand, disable hover
				card.hovered = false
				card.hoverOffset = 0
			} else {
				// Can't drop here, animate back to original position
				card.AnimateTo(card.originalX, card.originalY)
			}
			return
		}
	}
	
	// Check hand zone
	if e.handZone.Contains(x, y) {
		if e.CanDropCard(card, e.handZone) {
			// Check if card is from board - if so, move it back to hand
			if e.RemoveCardFromBoard(card) {
				e.hand = append(e.hand, card)
				// LayoutHand will be called after this
			}
			// If already in hand, LayoutHand will just reposition
		} else {
			// Can't drop here, animate back to original position
			card.AnimateTo(card.originalX, card.originalY)
		}
		return
	}
	
	// Not in any zone, animate back to original position
	card.AnimateTo(card.originalX, card.originalY)
}

func (e *CardGame) RemoveCardFromHand(card *Card) {
	for i, c := range e.hand {
		if c == card {
			// Remove card from hand
			e.hand = append(e.hand[:i], e.hand[i+1:]...)
			return
		}
	}
}

func (e *CardGame) RemoveCardFromBoard(card *Card) bool {
	for i, c := range e.board {
		if c == card {
			// Remove card from board
			e.board = append(e.board[:i], e.board[i+1:]...)
			return true
		}
	}
	return false
}

func (e *CardGame) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	// Update draw button
	if e.drawButton != nil {
		e.drawButton.Update(msg)
	}
	
	// Update cards in hand
	for _, c := range e.hand {
		c.Update(msg)
	}
	
	// Update cards on board (for animations)
	for _, c := range e.board {
		c.Update(msg)
	}
	
	return e, nil
}

func (e *CardGame) View() *ui.TileMap {
	screen := ui.NewTileMap(e.W, e.H, nil)
	
	// Draw drop zones
	zoneStyle := ui.NewStyle().
		Border(ui.Borders["rounded"]).
		BorderForeground(ui.Colors["dark-gray"]).
		Background(ui.Colors["dark"])
	
	for _, zone := range e.boardZones {
		zoneTile := ui.NewTileMap(zone.W, zone.H, nil)
		zoneTile = zoneStyle.Render(zoneTile)
		screen = screen.Overlay(zoneTile, zone.X, zone.Y)
	}
	
	// Draw hand zone
	handTile := ui.NewTileMap(e.handZone.W, e.handZone.H, nil)
	handTile = zoneStyle.Render(handTile)
	screen = screen.Overlay(handTile, e.handZone.X, e.handZone.Y)
	
	// Draw deck zone
	deckStyle := ui.NewStyle().
		Border(ui.Borders["rounded"]).
		BorderForeground(ui.Colors["blue"]).
		Background(ui.Colors["dark"])
	deckZone := ui.NewTileMap(11, 10, nil)
	deckZone = deckStyle.Render(deckZone)
	screen = screen.Overlay(deckZone, e.W-13, 3)
	
	// Draw deck count with better formatting
	deckCountStr := "Empty"
	if len(e.deck) > 0 {
		if len(e.deck) < 10 {
			deckCountStr = "Deck: " + string(rune('0'+len(e.deck)))
		} else {
			deckCountStr = "Deck: " + string(rune('0'+len(e.deck)/10)) + string(rune('0'+len(e.deck)%10))
		}
	}
	deckCount := ui.Text(deckCountStr)
	screen = screen.Overlay(deckCount.View(), e.W-12, 5)
	
	// Draw cards on board
	for _, c := range e.board {
		screen = screen.Overlay(c.View(), c.X, c.Y)
	}
	
	// Draw cards in hand - non-hovered first, then hovered/dragged on top
	var topCard *Card
	for _, c := range e.hand {
		if c.hovered || c.dragging {
			topCard = c
		} else {
			screen = screen.Overlay(c.View(), c.X, c.GetDrawY())
		}
	}
	// Draw hovered/dragged card on top
	if topCard != nil {
		screen = screen.Overlay(topCard.View(), topCard.X, topCard.GetDrawY())
	}
	
	// Draw button on top so it's always clickable
	if e.drawButton != nil {
		buttonView := e.drawButton.View()
		screen = screen.Overlay(buttonView, e.W-12, 15)
		
		// Draw hint text
		if len(e.deck) == 0 {
			hint := ui.Text("No cards left")
			screen = screen.Overlay(hint.View(), e.W-13, 18)
		}
	}
	
	return screen
}

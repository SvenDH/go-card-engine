package screens

import (
	"github.com/SvenDH/go-card-engine/tween"
	"github.com/SvenDH/go-card-engine/ui"
)

const (
	CardLocDeck = iota
	CardLocHand
	CardLocBoard
)

type CardImage struct {
	Name        string
	Picture     *ui.TileMap
	BorderColor ui.ColorIndex
	BorderStyle ui.Border
}

func (c *CardImage) Init() ui.Cmd {
	return nil
}

func (c *CardImage) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	return c, nil
}

func (c *CardImage) View() *ui.TileMap {
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
	card := ui.JoinVertical(ui.Left, top, pic)
	return borderStyle.Render(card)
}

type Card struct {
	X, Y     int
	Name     string
	Picture  *ui.TileMap
	Location int
	OnClick  ui.Cmd
	OnDrop   ui.Cmd

	cardImage                *CardImage
	input                    *ui.Zone
	dragging                 bool
	dragOffsetX, dragOffsetY int
	originalX, originalY     int
	// Animation
	tweenX    *tween.Tween
	tweenY    *tween.Tween
	animating bool
	// Visual state
	hoverOffset int
	hovered     bool
}

func (c *Card) Init() ui.Cmd {
	c.cardImage = &CardImage{
		Name:        c.Name,
		Picture:     c.Picture,
		BorderColor: ui.Colors["light-beige"],
		BorderStyle: ui.Borders["card"],
	}
	c.input = &ui.Zone{
		M:         c.cardImage,
		Draggable: true,
		Capture:   true,
		Enter: func(msg ui.Msg) ui.Cmd {
			if !c.dragging {
				c.hoverOffset = 2
				c.hovered = true
			}
			return nil
		},
		Leave: func(msg ui.Msg) ui.Cmd {
			if !c.dragging {
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

func (c *Card) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	switch m := msg.(type) {
	case ui.Tick:
		// Update tween animations
		if c.animating && !c.dragging && c.tweenX != nil && c.tweenY != nil {
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
		}
	}
	return c.input.Update(msg)
}

func (c *Card) AnimateTo(x, y int) {
	// Create new tweens for smooth animation
	duration := float32(0.3) // 0.3 seconds duration
	c.tweenX = tween.New(float32(c.X), float32(x), duration, tween.OutQuad)
	c.tweenY = tween.New(float32(c.Y), float32(y), duration, tween.OutQuad)
	c.animating = true
}

func (c *Card) View() *ui.TileMap {
	return c.input.View()
}

func (c *Card) GetDrawY() int {
	return c.Y - c.hoverOffset
}

type Lanes struct {
	Game      *CardGame
	zones     []ui.Model
	cards     []ui.Model
	cardStyle ui.Style
}

func (l *Lanes) Init() ui.Cmd {
	zoneStyle := ui.NewStyle().
		Margin(1).
		Border(ui.Borders["roundheavy"]).
		BorderForeground(ui.Colors["dark-brown"])

	l.zones = make([]ui.Model, 5)
	for i := range l.zones {
		laneIndex := i // Capture loop variable
		l.zones[i] = &ui.Zone{
			M:         zoneStyle.Render(ui.NewTileMap(10, 12, nil)),
			Droppable: true,
			Drop: func(msg ui.Msg) ui.Cmd {
				// Get dragged card from zone's DragData
				event := msg.(ui.MouseEvent)
				if event.Zone != nil && event.Zone.DragData != nil {
					card := event.Zone.DragData.(*Card)
					zone := l.zones[laneIndex].(*ui.Zone)
					if l.Game.CanDropCard(card, zone) {
						// Remove from hand and add to board
						wasInHand := card.Location == CardLocHand
						l.Game.RemoveCard(card)
						card.Location = CardLocBoard
						l.cards[laneIndex] = card
						// Animate to zone center (accounting for card border width of 2)
						targetX := zone.X + (zone.W-card.Picture.W-2)/2
						targetY := zone.Y + (zone.H-card.Picture.H-2)/2
						card.AnimateTo(targetX, targetY)
						// Card is no longer in hand, disable hover
						card.hovered = false
						card.hoverOffset = 0
						// Rearrange remaining cards in hand if this card came from hand
						if wasInHand {
							l.Game.LayoutHand()
						}
					} else {
						// Can't drop here, animate back to original position
						card.AnimateTo(card.originalX, card.originalY)
					}
				}
				return nil
			},
		}
	}
	l.cards = make([]ui.Model, 5)
	l.cardStyle = ui.NewStyle().Margin(1)
	return nil
}

func (l *Lanes) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	var cmd ui.Cmd
	var cmds []ui.Cmd
	for i, zone := range l.zones {
		l.zones[i], cmd = zone.Update(msg)
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

func (l *Lanes) View() *ui.TileMap {
	// Render all zone backgrounds
	tms := make([]*ui.TileMap, len(l.zones))
	for i, zone := range l.zones {
		tms[i] = zone.View()
	}
	return ui.JoinHorizontal(ui.Top, tms...)
}

func (l *Lanes) Remove(card *Card) bool {
	for i, c := range l.cards {
		if c.(*Card) == card {
			// Remove card from board
			l.cards = append(l.cards[:i], l.cards[i+1:]...)
			return true
		}
	}
	return false
}

type CardGame struct {
	W, H               int
	deck               []ui.Model
	hand               []ui.Model
	handZone           *ui.Zone
	drawButton         *ui.Zone
	playerLanes        ui.Model
	enemyLanes         ui.Model
	playerResources    []ui.Model
	enemyResources     []ui.Model
	playerResourceZone *ui.Zone
	enemyResourceZone  *ui.Zone
}

func NewCardGame(width, height int) *CardGame {
	e := &CardGame{
		W:               width,
		H:               height,
		deck:            make([]ui.Model, 0),
		hand:            make([]ui.Model, 0),
		playerLanes:     &Lanes{},
		enemyLanes:      &Lanes{},
		playerResources: make([]ui.Model, 0),
		enemyResources:  make([]ui.Model, 0),
	}
	e.playerLanes.(*Lanes).Game = e
	e.enemyLanes.(*Lanes).Game = e

	tm := ui.NewStyle().
		Border(ui.Borders["rounded"]).
		BorderForeground(ui.Colors["dark-gray"]).
		Background(ui.Colors["dark"]).
		Render(ui.NewTileMap(width-15, 12, nil))

	e.handZone = &ui.Zone{
		M:         tm,
		W:         tm.W,
		H:         tm.H,
		Droppable: true,
		Drop: func(msg ui.Msg) ui.Cmd {
			// Handle dropping cards back to hand
			event := msg.(ui.MouseEvent)
			if event.Zone != nil && event.Zone.DragData != nil {
				card := event.Zone.DragData.(*Card)
				if e.CanDropCard(card, e.handZone) {
					// Check if card is from board - if so, move it back to hand
					if e.RemoveCard(card) {
						card.Location = CardLocHand
						e.hand = append(e.hand, card)
						e.LayoutHand()
					}
				} else {
					// Can't drop here, animate back to original position
					card.AnimateTo(card.originalX, card.originalY)
				}
			}
			return nil
		},
	}

	// Create player resource zone
	playerResourceTM := ui.NewStyle().
		Border(ui.Borders["rounded"]).
		BorderForeground(ui.Colors["green"]).
		Background(ui.Colors["dark"]).
		Render(ui.NewTileMap(width-15, 14, nil))

	e.playerResourceZone = &ui.Zone{
		M:         playerResourceTM,
		W:         playerResourceTM.W,
		H:         playerResourceTM.H,
		Droppable: true,
		Drop: func(msg ui.Msg) ui.Cmd {
			// Handle dropping resource cards
			event := msg.(ui.MouseEvent)
			if event.Zone != nil && event.Zone.DragData != nil {
				card := event.Zone.DragData.(*Card)
				// Remove from previous location
				e.RemoveCard(card)
				// Add to player resources
				card.Location = CardLocBoard
				e.playerResources = append(e.playerResources, card)
				// Animate to resource zone
				card.AnimateTo(e.playerResourceZone.X+2, e.playerResourceZone.Y+2)
			}
			return nil
		},
	}

	// Create enemy resource zone
	enemyResourceTM := ui.NewStyle().
		Border(ui.Borders["rounded"]).
		BorderForeground(ui.Colors["red"]).
		Background(ui.Colors["dark"]).
		Render(ui.NewTileMap(width-15, 14, nil))

	e.enemyResourceZone = &ui.Zone{
		M:         enemyResourceTM,
		W:         enemyResourceTM.W,
		H:         enemyResourceTM.H,
		Droppable: true,
		Drop: func(msg ui.Msg) ui.Cmd {
			// Handle dropping resource cards
			event := msg.(ui.MouseEvent)
			if event.Zone != nil && event.Zone.DragData != nil {
				card := event.Zone.DragData.(*Card)
				// Remove from previous location
				e.RemoveCard(card)
				// Add to enemy resources
				card.Location = CardLocBoard
				e.enemyResources = append(e.enemyResources, card)
				// Animate to resource zone
				card.AnimateTo(e.enemyResourceZone.X+2, e.enemyResourceZone.Y+2)
			}
			return nil
		},
	}

	return e
}

func (e *CardGame) Init() ui.Cmd {
	e.playerLanes.Init()
	e.enemyLanes.Init()

	img, _ := ui.LoadTileMap("assets/frog10x12.txt")

	// Create deck with 15 cards
	for range 15 {
		card := &Card{
			X:        e.W - 13,
			Y:        3,
			Name:     "Frog",
			Picture:  img,
			Location: CardLocDeck,
		}
		card.OnClick = func() ui.Msg {
			return nil
		}
		card.OnDrop = func() ui.Msg {
			// Always animate back to original position as fallback
			// If a zone's Drop handler accepts this card, it will call AnimateTo
			// with a new target, which will override this animation
			card.AnimateTo(card.originalX, card.originalY)
			return nil
		}
		card.Init()
		e.deck = append(e.deck, card)
	}

	// Create draw button
	e.drawButton = &ui.Zone{
		M: ui.Text("DRAW"),
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
		card := e.deck[len(e.deck)-1].(*Card)
		e.deck = e.deck[:len(e.deck)-1]

		// Add to hand first
		card.Location = CardLocHand
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
	cardWidth := e.hand[0].(*Card).Picture.W + 2 // +2 for border

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
		// Always create new animation - this will override any existing tween
		card.(*Card).AnimateTo(targetX, targetY)
	}
}

func (e *CardGame) CanDropCard(card *Card, zone *ui.Zone) bool {
	// For now, always return true
	return true
}

func (e *CardGame) InAnyDropZone(card *Card) bool {
	// Check if card is over handZone or any lane zone
	x, y := card.X/ui.TileSize, card.Y/ui.TileSize

	if e.handZone.InBounds(x, y) {
		return true
	}

	// Check player lanes
	if lanes, ok := e.playerLanes.(*Lanes); ok {
		for _, zone := range lanes.zones {
			if z, ok := zone.(*ui.Zone); ok && z.InBounds(x, y) {
				return true
			}
		}
	}

	// Check enemy lanes
	if lanes, ok := e.enemyLanes.(*Lanes); ok {
		for _, zone := range lanes.zones {
			if z, ok := zone.(*ui.Zone); ok && z.InBounds(x, y) {
				return true
			}
		}
	}

	// Check resource zones
	if e.playerResourceZone != nil && e.playerResourceZone.InBounds(x, y) {
		return true
	}
	if e.enemyResourceZone != nil && e.enemyResourceZone.InBounds(x, y) {
		return true
	}

	return false
}

func (e *CardGame) TryDropCard(card *Card, x, y int) {
	// Check hand zone
	if e.handZone.InBounds(x, y) {
		if e.CanDropCard(card, e.handZone) {
			// Check if card is from board - if so, move it back to hand
			if e.RemoveCard(card) {
				e.hand = append(e.hand, card)
				// LayoutHand will be called after this
			}
			// If already in hand, LayoutHand will just reposition
			return
		}
	}
	// Not in any zone, animate back to original position
	card.AnimateTo(card.originalX, card.originalY)
}

func (e *CardGame) RemoveCard(card *Card) bool {
	// Remove from hand
	if card.Location == CardLocHand {
		for i, c := range e.hand {
			if c == card {
				e.hand = append(e.hand[:i], e.hand[i+1:]...)
				return true
			}
		}
	}

	// Remove from board
	if card.Location == CardLocBoard {
		if lanes, ok := e.playerLanes.(*Lanes); ok {
			for i, c := range lanes.cards {
				if c != nil && c.(*Card) == card {
					lanes.cards[i] = nil
					return true
				}
			}
		}
		if lanes, ok := e.enemyLanes.(*Lanes); ok {
			for i, c := range lanes.cards {
				if c != nil && c.(*Card) == card {
					lanes.cards[i] = nil
					return true
				}
			}
		}
		// Check player resources
		for i, c := range e.playerResources {
			if c.(*Card) == card {
				e.playerResources = append(e.playerResources[:i], e.playerResources[i+1:]...)
				return true
			}
		}
		// Check enemy resources
		for i, c := range e.enemyResources {
			if c.(*Card) == card {
				e.enemyResources = append(e.enemyResources[:i], e.enemyResources[i+1:]...)
				return true
			}
		}
	}

	return false
}

func (e *CardGame) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	var cmd ui.Cmd
	var cmds []ui.Cmd
	// Update draw button
	if e.drawButton != nil {
		_, cmd := e.drawButton.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update cards in hand
	for _, c := range e.hand {
		_, cmd := c.Update(msg)
		cmds = append(cmds, cmd)
	}

	e.playerLanes, cmd = e.playerLanes.Update(msg)
	cmds = append(cmds, cmd)

	e.enemyLanes, cmd = e.enemyLanes.Update(msg)
	cmds = append(cmds, cmd)

	// Update resource zones
	if e.playerResourceZone != nil {
		_, cmd = e.playerResourceZone.Update(msg)
		cmds = append(cmds, cmd)
	}
	if e.enemyResourceZone != nil {
		_, cmd = e.enemyResourceZone.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update resource cards
	for _, c := range e.playerResources {
		_, cmd = c.Update(msg)
		cmds = append(cmds, cmd)
	}
	for _, c := range e.enemyResources {
		_, cmd = c.Update(msg)
		cmds = append(cmds, cmd)
	}

	return e, ui.Batch(cmds...)
}

func (e *CardGame) View() *ui.TileMap {
	screen := ui.NewTileMap(e.W, e.H, nil)

	// Draw hand zone
	screen = screen.Overlay(e.handZone.View(), 0, e.H-12)

	// Draw enemy lanes at the top
	screen = screen.Overlay(ui.JoinVertical(
		ui.Center,
		e.enemyResourceZone.View(),
		e.enemyLanes.View(),
		e.playerLanes.View(),
		e.playerResourceZone.View(),
	), 0, 0)

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

	for _, card := range e.enemyResources {
		if card != nil {
			c := card.(*Card)
			if !c.dragging {
				screen = screen.Overlay(c.View(), c.X, c.Y)
			}
		}
	}

	for _, card := range e.enemyLanes.(*Lanes).cards {
		if card != nil {
			c := card.(*Card)
			if !c.dragging {
				screen = screen.Overlay(c.View(), c.X, c.Y)
			}
		}
	}

	// Draw cards in player lanes at their absolute positions
	for _, card := range e.playerLanes.(*Lanes).cards {
		if card != nil {
			c := card.(*Card)
			// Skip if card is being dragged (will be drawn on top later)
			if !c.dragging {
				screen = screen.Overlay(c.View(), c.X, c.Y)
			}
		}
	}
	// Draw player resource cards
	for _, card := range e.playerResources {
		if card != nil {
			c := card.(*Card)
			if !c.dragging {
				screen = screen.Overlay(c.View(), c.X, c.Y)
			}
		}
	}

	// Draw cards in hand - non-hovered first, then hovered/dragged on top
	var topCard *Card
	for _, c := range e.hand {
		if c.(*Card).hovered || c.(*Card).dragging {
			topCard = c.(*Card)
		} else {
			screen = screen.Overlay(c.View(), c.(*Card).X, c.(*Card).GetDrawY())
		}
	}
	// Draw hovered/dragged card on top (from hand or lanes)
	if topCard != nil {
		screen = screen.Overlay(topCard.View(), topCard.X, topCard.GetDrawY())
	}

	// Also draw any dragged cards from lanes on top
	if lanes, ok := e.playerLanes.(*Lanes); ok {
		for _, card := range lanes.cards {
			if card != nil {
				c := card.(*Card)
				if c.dragging {
					screen = screen.Overlay(c.View(), c.X, c.Y)
				}
			}
		}
	}
	if lanes, ok := e.enemyLanes.(*Lanes); ok {
		for _, card := range lanes.cards {
			if card != nil {
				c := card.(*Card)
				if c.dragging {
					screen = screen.Overlay(c.View(), c.X, c.Y)
				}
			}
		}
	}

	// Also draw any dragged resource cards on top
	for _, card := range e.playerResources {
		if card != nil {
			c := card.(*Card)
			if c.dragging {
				screen = screen.Overlay(c.View(), c.X, c.Y)
			}
		}
	}
	for _, card := range e.enemyResources {
		if card != nil {
			c := card.(*Card)
			if c.dragging {
				screen = screen.Overlay(c.View(), c.X, c.Y)
			}
		}
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

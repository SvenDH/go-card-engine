package screens

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/SvenDH/go-card-engine/game"
	"github.com/SvenDH/go-card-engine/ui"
)

var cardsParser = game.NewCardParser()

var cardData []byte
var cards []*game.Card

func init() {
	var err error
	cardData, err = os.ReadFile("assets/cards.txt")
	if err != nil {
		panic(err)
	}
	for _, txt := range strings.Split(string(cardData), "\n\n") {
		card, err := cardsParser.Parse(txt, true)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(card)
		cards = append(cards, card)
	}
}

type CardGame struct {
	W, H int

	hand                    *Hand
	playerLanes             ui.Model
	enemyLanes              ui.Model
	playerResources         *ResourceZone
	enemyResources          *ResourceZone
	skipButton              *ui.Zone
	skipButtonHovered       bool
	prompting               bool
	phaseLabel              *ui.Zone
	currentPhase            string
	currentPlayer           string
	validFields             map[int]bool
	selectedCard            *Card
	enemySelectedCard       *Card
	playableCards           []*Card
	abilityMenu             []*ui.Zone
	abilityMenuHovered      []bool
	abilityChoices          []any
	promptingAbility        bool
	stack                   *Stack
	playerLife              int
	enemyLife               int

	gameState *game.GameState
	player    *game.Player
	enemy     *game.Player
	cardMap   map[int]*Card
}

func NewCardGame(width, height int) *CardGame {
	e := &CardGame{
		W:           width,
		H:           height,
		playerLanes: &Lanes{},
		enemyLanes:  &Lanes{},
		cardMap:     make(map[int]*Card),
	}
	e.playerLanes.(*Lanes).Game = e
	e.playerLanes.(*Lanes).isPlayer = true
	e.enemyLanes.(*Lanes).Game = e
	e.enemyLanes.(*Lanes).isPlayer = false

	e.hand = NewHand(e, width, height)
	e.playerResources = NewResourceZone(e, width, height, true)
	e.enemyResources = NewResourceZone(e, width, height, false)

	// Create skip button
	e.skipButton = &ui.Zone{
		M:       ui.NewTileMap(8, 3, nil),
		W:       8,
		H:       3,
		Capture: true,
		Enter: func(msg ui.Msg) ui.Cmd {
			e.skipButtonHovered = true
			return nil
		},
		Leave: func(msg ui.Msg) ui.Cmd {
			e.skipButtonHovered = false
			return nil
		},
		Click: func(msg ui.Msg) ui.Cmd {
			if e.prompting && e.player != nil {
				// Send skip code to player
				e.player.Send(game.Msg{Selected: []int{game.SkipCode}})
				e.prompting = false
				e.promptingAbility = false
				e.abilityMenu = []*ui.Zone{}
				e.abilityMenuHovered = []bool{}
				e.abilityChoices = []any{}
			}
			return nil
		},
	}

	// Create phase label
	e.phaseLabel = &ui.Zone{
		M: ui.NewTileMap(20, 3, nil),
		W: 20,
		H: 3,
	}
	e.currentPhase = "Start"
	e.currentPlayer = "Player"
	e.playerLife = 20
	e.enemyLife = 20
	e.cardMap = make(map[int]*Card)
	e.validFields = make(map[int]bool)
	e.abilityMenu = []*ui.Zone{}
	e.abilityMenuHovered = []bool{}
	e.abilityChoices = []any{}
	e.promptingAbility = false

	// Create stack zone
	e.stack = NewStack(e)

	return e
}

func (e *CardGame) Init() ui.Cmd {
	// Initialize random seed for bot
	rand.Seed(time.Now().UnixNano())

	e.playerLanes.Init()
	e.enemyLanes.Init()

	// Deck is no longer pre-created - cards are created on-demand when drawn

	go e.StartGame()

	return nil
}

func (e *CardGame) StartGame() {
	e.gameState = game.NewGame()
	e.player = e.gameState.AddPlayer(cards...)
	e.enemy = e.gameState.AddPlayer(cards...)
	e.gameState.On(game.AllEvents, e.eventHandler)
	e.gameState.Run()
}

func (e *CardGame) CreateCard(cardInstance *game.CardInstance) *Card {
	img, _ := ui.LoadTileMap("assets/frog10x12.txt")
	// Position cards at deck location (top-right)
	deckX := e.W - 13
	deckY := 3
	card := &Card{
		X:            deckX,
		Y:            deckY,
		Name:         cardInstance.Card.Name,
		Picture:      img,
		Location:     CardLocDeck,
		game:         e,
		cardInstance: cardInstance,
	}
	// Set original position for animations
	card.originalX = card.X
	card.originalY = card.Y
	card.OnClick = func() ui.Msg {
		return nil
	}
	card.OnDrop = func() ui.Msg {
		card.AnimateTo(card.originalX, card.originalY)
		return nil
	}
	card.Init()
	e.cardMap[cardInstance.GetId()] = card
	return card
}

func (e *CardGame) Draw(cardInstance *game.CardInstance) {
	// Create new card at deck location
	card := e.CreateCard(cardInstance)

	// Add to hand
	e.hand.AddCard(card)
}

func (e *CardGame) PromptCard(choices []any) {
	for _, card := range e.hand.Cards() {
		card.(*Card).Highlighted = false
	}
	e.playableCards = []*Card{}
	for _, choice := range choices {
		if card, ok := e.cardMap[choice.(*game.CardInstance).GetId()]; ok {
			card.Highlighted = true
			e.playableCards = append(e.playableCards, card)
		}
	}
}

func (e *CardGame) PromptField(choices []any) {
	// Clear previous valid fields
	e.validFields = make(map[int]bool)
	for _, choice := range choices {
		if fieldIdx, ok := choice.(int); ok {
			e.validFields[fieldIdx] = true
		}
	}
}

func (e *CardGame) PromptAbility(choices []any) {
	// Store choices and create menu zones
	e.abilityChoices = choices
	e.abilityMenu = make([]*ui.Zone, len(choices))
	e.abilityMenuHovered = make([]bool, len(choices))

	for i := range choices {
		choiceIndex := i // Capture loop variable
		e.abilityMenu[i] = &ui.Zone{
			M:       ui.NewTileMap(30, 3, nil),
			W:       30,
			H:       3,
			Capture: true,
			Enter: func(msg ui.Msg) ui.Cmd {
				e.abilityMenuHovered[choiceIndex] = true
				return nil
			},
			Leave: func(msg ui.Msg) ui.Cmd {
				e.abilityMenuHovered[choiceIndex] = false
				return nil
			},
			Click: func(msg ui.Msg) ui.Cmd {
				if e.promptingAbility && e.player != nil {
					// Send ability selection to player
					go func() {
						e.player.Send(game.Msg{Selected: []int{choiceIndex}})
					}()
					e.prompting = false
					e.promptingAbility = false
					e.abilityMenu = []*ui.Zone{}
					e.abilityMenuHovered = []bool{}
					e.abilityChoices = []any{}
				}
				return nil
			},
		}
	}
}

func (e *CardGame) eventHandler(event *game.Event) {
	player := event.Player

	// Handle card reveal events
	if event.Event == game.EventOnDraw || event.Event == game.EventOnEnterBoard {
		// TODO: handle card reveal events
	}
	fmt.Println(event)
	switch event.Event {
	case game.EventOnDraw:
		if player == e.player {
			e.Draw(event.Args[0].(*game.CardInstance))
		} else {
			// Draw card for enemy
			cardInstance := event.Args[0].(*game.CardInstance)
			if _, exists := e.cardMap[cardInstance.GetId()]; !exists {
				e.CreateCard(cardInstance)
			}
		}
	case game.EventOnEnterBoard:
		// Move card from stack to actual field position
		if len(event.Args) >= 2 {
			cardInstance := event.Args[0].(*game.CardInstance)
			fieldIndex := event.Args[1].(int)

			// Create card if it doesn't exist (for enemy cards)
			card, ok := e.cardMap[cardInstance.GetId()]
			if !ok {
				card = e.CreateCard(cardInstance)
				ok = true
			}

			// For ALL enemy cards entering the board, position at stack first
			if ok && player != e.player {
				stackX := e.W/2 - 6
				stackY := e.H/2 - 8
				card.X = stackX + 1
				card.Y = stackY + 1
				card.originalX = card.X
				card.originalY = card.Y
			}

			if ok {
				card.Location = CardLocBoard

				// Determine which lanes to use
				var lanes *Lanes
				if player == e.player {
					lanes = e.playerLanes.(*Lanes)
				} else {
					lanes = e.enemyLanes.(*Lanes)
				}

				// Place card in the lane
				lanes.cards[fieldIndex] = card
				zone := lanes.zones[fieldIndex].zone

				// Calculate target position
				targetX := zone.X + (zone.W-card.Picture.W-2)/2
				targetY := zone.Y + (zone.H-card.Picture.H-2)/2 - 1

				// Update original position for this card's new home
				card.originalX = targetX
				card.originalY = targetY

				// Animate to zone center
				card.AnimateTo(targetX, targetY)

				// Remove highlight when leaving stack
				card.Highlighted = false

				// Clear stack if this was the stacked card
				if e.stack.GetCard() == card {
					e.stack.Clear()
				}
				// Clear enemy selected card if this was it
				if e.enemySelectedCard == card {
					e.enemySelectedCard = nil
				}
			}
		}
	case game.EventAtStartPhase:
		e.currentPhase = "Start"
		e.updateCurrentPlayer(player)
	case game.EventAtDrawPhase:
		e.currentPhase = "Draw"
		e.updateCurrentPlayer(player)
	case game.EventAtPlayPhase:
		e.currentPhase = "Play"
		e.updateCurrentPlayer(player)
	case game.EventAtEndPhase:
		e.currentPhase = "End"
		e.updateCurrentPlayer(player)
	case game.EventOnGainLife:
		if len(event.Args) > 0 {
			amount := event.Args[0].(int)
			if player == e.player {
				e.playerLife += amount
			} else {
				e.enemyLife += amount
			}
		}
	case game.EventOnLoseLife:
		if len(event.Args) > 0 {
			amount := event.Args[0].(int)
			if player == e.player {
				e.playerLife -= amount
			} else {
				e.enemyLife -= amount
			}
		}
	case game.EventOnAttack:
		// Trigger attack bump animation for the attacking card
		if len(event.Args) > 0 {
			if cardInst, ok := event.Args[0].(*game.CardInstance); ok {
				if card, exists := e.cardMap[cardInst.GetId()]; exists {
					// Direction: 1 for player attacking down, -1 for enemy attacking up
					direction := 1
					if player != e.player {
						direction = -1
					}
					card.AnimateBump(direction)
				}
			}
		}
	case game.EventPromptCard:
		if player == e.player {
			e.prompting = true
			e.PromptCard(event.Args[1:])
		} else {
			e.enemyBotPromptCard(event.Args[1:], player)
		}
	case game.EventPromptField:
		if player == e.player {
			e.PromptField(event.Args[1:])
		} else {
			// Put enemy card on stack - use the tracked enemy selected card
			if e.enemySelectedCard != nil {
				// Set card on stack (fieldIndex will be determined by bot)
				e.stack.SetCard(e.enemySelectedCard, 0)
			}
			e.enemyBotPromptField(event.Args[1:])
		}
	case game.EventPromptAbility:
		if player == e.player {
			e.prompting = true
			e.promptingAbility = true
			e.PromptAbility(event.Args[1:])
		} else {
			e.enemyBotPromptAbility(event.Args[1:])
		}
	case game.EventPromptTarget:
		if player == e.player {
			e.prompting = true
		} else {
			e.enemyBotPromptTarget(event.Args[1:])
		}
	case game.EventPromptSource:
		if player == e.player {
			e.prompting = true
		} else {
			e.enemyBotPromptSource(event.Args[1:])
		}
	case game.EventPromptDiscard:
		if player == e.player {
			e.prompting = true
		} else {
			e.enemyBotPromptDiscard(event.Args[1:])
		}
	}
}

func (e *CardGame) updateCurrentPlayer(player *game.Player) {
	if player == e.player {
		e.currentPlayer = "Player"
	} else if player == e.enemy {
		e.currentPlayer = "Enemy"
	} else {
		e.currentPlayer = "Unknown"
	}
}

// LayoutHand delegates to the hand component
func (e *CardGame) LayoutHand() {
	e.hand.Layout()
}

// Helper to find any currently dragged card
func (e *CardGame) getDraggedCard() *Card {
	// Check hand
	for _, c := range e.hand.Cards() {
		if c.(*Card).dragging {
			return c.(*Card)
		}
	}
	// Check all card collections
	allCollections := [][]ui.Model{
		e.playerResources.Cards(),
		e.enemyResources.Cards(),
		e.playerLanes.(*Lanes).cards,
		e.enemyLanes.(*Lanes).cards,
	}
	for _, cards := range allCollections {
		for _, c := range cards {
			if c != nil && c.(*Card).dragging {
				return c.(*Card)
			}
		}
	}
	return nil
}

// Update zone hover states based on dragged card position
func (e *CardGame) updateDragHoverStates() {
	// Zones now handle their own hover state through Enter/Leave events
	// This method can be simplified or removed
}

func (e *CardGame) CanDropCard(card *Card, fieldIndex int) bool {
	// Only allow dropping selected card in valid fields
	if e.selectedCard != nil && e.selectedCard == card {
		if len(e.validFields) > 0 {
			return e.validFields[fieldIndex]
		}
	}
	return false
}

// Helper to remove card from a collection and optionally relayout
func removeCardFromCollection(card *Card, collection []ui.Model, onRemove func()) ([]ui.Model, bool) {
	for i, c := range collection {
		if c.(*Card) == card {
			result := append(collection[:i], collection[i+1:]...)
			if onRemove != nil {
				onRemove()
			}
			return result, true
		}
	}
	return collection, false
}

func (e *CardGame) RemoveCard(card *Card) bool {
	// Remove from hand
	if card.Location == CardLocHand {
		return e.hand.RemoveCard(card)
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
		if e.playerResources.RemoveCard(card) {
			return true
		}

		// Check enemy resources
		if e.enemyResources.RemoveCard(card) {
			return true
		}
	}

	return false
}

// Helper to update a collection of models and collect commands
func updateModels(msg ui.Msg, models []ui.Model) []ui.Cmd {
	var cmds []ui.Cmd
	for _, m := range models {
		if m != nil {
			_, cmd := m.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

// Helper to update a single zone and return command
func updateZone(msg ui.Msg, zone *ui.Zone) ui.Cmd {
	if zone != nil {
		_, cmd := zone.Update(msg)
		return cmd
	}
	return nil
}

func (e *CardGame) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	var cmds []ui.Cmd

	// Update skip button
	if cmd := updateZone(msg, e.skipButton); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update ability menu
	for _, zone := range e.abilityMenu {
		if cmd := updateZone(msg, zone); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update phase label
	if cmd := updateZone(msg, e.phaseLabel); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update resource zones
	if _, cmd := e.playerResources.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if _, cmd := e.enemyResources.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update lanes
	var cmd ui.Cmd
	e.playerLanes, cmd = e.playerLanes.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	e.enemyLanes, cmd = e.enemyLanes.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update hand and all cards
	if _, cmd := e.hand.Update(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, updateModels(msg, e.hand.Cards())...)
	cmds = append(cmds, updateModels(msg, e.playerResources.Cards())...)
	cmds = append(cmds, updateModels(msg, e.enemyResources.Cards())...)

	// Update zone hover states when dragging
	e.updateDragHoverStates()

	return e, ui.Batch(cmds...)
}

func (e *CardGame) View() *ui.TileMap {
	screen := ui.NewTileMap(e.W, e.H, nil)

	// Draw hand zone (positioned to cover raised cards)
	screen = screen.Overlay(e.hand.View(), 0, e.H-10)

	// Draw enemy lanes at the top with resource zones
	screen = screen.Overlay(ui.JoinVertical(
		ui.Center,
		renderZoneWithHover(e.enemyResources.Zone(), e.enemyResources.IsHovered(), ui.Borders["round"], ui.Colors["dark-brown"], ui.Colors["brown"]),
		e.enemyLanes.View(),
		e.playerLanes.View(),
		renderZoneWithHover(e.playerResources.Zone(), e.playerResources.IsHovered(), ui.Borders["round"], ui.Colors["dark-brown"], ui.Colors["brown"]),
	), 0, 0)

	// Render UI components
	screen = e.renderDeckCount(screen)
	screen = e.renderLifeTotals(screen)
	screen = e.renderCards(screen)
	screen = e.renderStack(screen)
	screen = e.renderPhaseLabel(screen)
	screen = e.renderAbilityMenu(screen)
	screen = e.renderSkipButton(screen)

	return screen
}

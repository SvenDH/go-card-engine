package screens

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SvenDH/go-card-engine/engine"
	"github.com/SvenDH/go-card-engine/tween"
	"github.com/SvenDH/go-card-engine/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

var cardsParser = engine.NewCardParser()

var cardData []byte
var cards []*engine.Card

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

	hand               *Hand
	playerLanes        ui.Model
	enemyLanes         ui.Model
	playerResources    *ResourceZone
	enemyResources     *ResourceZone
	skipButton         *ui.Zone
	skipButtonHovered  bool
	prompting          bool
	phaseLabel         *ui.Zone
	currentPhase       string
	currentPlayer      string
	validFields        map[int]bool
	selectedCard       *Card
	enemySelectedCard  *Card
	playableCards      []*Card
	playableFields     []int
	cardChoices        []any // Raw choices from PromptCard
	abilityMenu        []*ui.Zone
	abilityMenuHovered []bool
	abilityChoices     []any
	abilityCard        *Card
	promptingAbility   bool
	targetChoices      []any   // Available targets for selection
	targetableCards    []*Card // Cards that can be targeted
	targetableFields   []int   // Fields that can be targeted (for empty slots or players)
	promptingTarget    bool
	stack              *Stack
	playerLife         int
	enemyLife          int
	playerEssence      map[string]int
	enemyEssence       map[string]int
	playerPile         []ui.Model
	enemyPile          []ui.Model

	// Screen shake state
	shakeX        int
	shakeY        int
	shakeSequence *tween.Sequence
	shaking       bool

	// Health flash state
	playerHealthFlash time.Time
	enemyHealthFlash  time.Time

	// Attack target visualization
	attackTargetField     int
	attackTargetTime      time.Time
	attackTargetIsPreview bool // true for hover preview, false for post-attack animation

	// Keyboard focus state
	focusMode         string // "hand", "field", "enemy-field", "ability-menu"
	focusHandIndex    int    // Index of focused card in hand
	focusFieldIndex   int    // Index of focused field/card
	focusAbilityIndex int    // Index of focused ability in menu

	gameState *engine.GameState
	player    *engine.Player
	enemy     *engine.Player
	cardMap   map[int]*Card

	// Mutex to protect shared state from concurrent access
	mu sync.RWMutex
}

func NewCardGame(width, height int) *CardGame {
	e := &CardGame{
		W:                     width,
		H:                     height,
		playerLanes:           &Lanes{},
		enemyLanes:            &Lanes{},
		cardMap:               make(map[int]*Card),
		playerPile:            make([]ui.Model, 0),
		enemyPile:             make([]ui.Model, 0),
		attackTargetField:     -1,
		attackTargetIsPreview: false,
		focusMode:             "", // No focus initially
		focusHandIndex:        0,
		focusFieldIndex:       0,
	}
	e.playerLanes.(*Lanes).Game = e
	e.playerLanes.(*Lanes).isPlayer = true
	e.enemyLanes.(*Lanes).Game = e
	e.enemyLanes.(*Lanes).isPlayer = false

	e.hand = NewHand(e, width, height)
	// Don't raise hand initially - only on keyboard focus
	e.playerResources = NewResourceZone(e, width, height, true)
	e.enemyResources = NewResourceZone(e, width, height, false)

	// Create skip button
	e.skipButton = &ui.Zone{
		M:       ui.NewImage(8, 3, nil),
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
				e.player.Send(engine.Msg{Selected: []int{engine.SkipCode}})
				e.prompting = false
				e.promptingAbility = false
				e.abilityMenu = []*ui.Zone{}
				e.abilityMenuHovered = []bool{}
				e.abilityChoices = []any{}
				e.abilityCard = nil
				// Clear attack target preview
				e.attackTargetField = -1
				e.attackTargetIsPreview = false
				// Re-enable all cards
				for _, card := range e.hand.Cards() {
					card.(*Card).Disabled = false
				}
			}
			return nil
		},
	}

	// Create phase label
	e.phaseLabel = &ui.Zone{
		M: ui.NewImage(20, 3, nil),
		W: 20,
		H: 3,
	}
	e.currentPhase = "Start"
	e.currentPlayer = "Player"
	e.playerLife = 20
	e.enemyLife = 20
	e.playerEssence = make(map[string]int)
	e.enemyEssence = make(map[string]int)
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
	e.gameState = engine.NewGame()
	e.player = e.gameState.AddPlayer(cards...)
	e.enemy = e.gameState.AddPlayer(cards...)
	e.gameState.On(engine.AllEvents, e.eventHandler)
	e.gameState.Run()
}

func (e *CardGame) CreateCard(cardInstance *engine.CardInstance) *Card {
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
	card.Init()
	e.cardMap[cardInstance.GetId()] = card
	return card
}

func (e *CardGame) Draw(cardInstance *engine.CardInstance) {
	// Create new card at deck location
	card := e.CreateCard(cardInstance)

	// Add to hand
	e.hand.Add(card)
}

func (e *CardGame) PromptCard(choices []any) {
	e.playableCards = nil
	e.playableFields = nil
	e.cardChoices = choices // Store raw choices for later reference
	// Collect playable cards first
	for _, choice := range choices {
		// Skip non-card choices (like skip code integers)
		if cardInst, ok := choice.(*engine.CardInstance); ok {
			if card, exists := e.cardMap[cardInst.GetId()]; exists {
				e.playableCards = append(e.playableCards, card)
			}
		}
	}

	// Only disable cards if there are actual card choices
	if len(e.playableCards) > 0 {
		// Disable all cards in hand first
		e.disableHandCards()
		// Enable only the playable cards
		for _, card := range e.playableCards {
			card.Disabled = false
		}
		// Switch focus to hand when prompting for cards (only if keyboard focus is active)
		if e.focusMode != "" {
			e.focusMode = "hand"
			e.hand.hovered = true
			e.LayoutHand()
			handCards := e.hand.Cards()
			e.focusHandIndex = 0
			// Find first enabled card
			for i, card := range handCards {
				if !card.(*Card).Disabled {
					e.focusHandIndex = i
					break
				}
			}
		}
	} else {
		// No card choices - enable all cards and switch to field for ability activation
		e.enableHandCards()
		// Switch to field mode for ability activation if we have choices (only if keyboard focus is active)
		if len(choices) > 0 && e.focusMode != "" {
			e.focusMode = "field"
			e.focusFieldIndex = 0
			// Lower hand when switching to field
			e.hand.hovered = false
			e.LayoutHand()
		}
	}
}

func (e *CardGame) PromptField(choices []any) {
	e.playableCards = nil
	e.cardChoices = nil // Clear card choices when switching to field prompt
	// Re-enable all cards when switching to field prompt
	e.enableHandCards()
	// Clear previous valid fields
	e.validFields = make(map[int]bool)
	for _, choice := range choices {
		if fieldIdx, ok := choice.(int); ok {
			e.validFields[fieldIdx] = true
		}
	}
	e.playableFields = make([]int, len(choices))
	for i, j := range choices {
		e.playableFields[i] = j.(int)
	}

	// Switch focus to field mode and set to first valid field (only if keyboard focus is active)
	if len(e.playableFields) > 0 && e.focusMode != "" {
		e.focusMode = "field"
		e.focusFieldIndex = e.playableFields[0]
		// Lower hand when switching to field
		e.hand.hovered = false
		e.LayoutHand()
	}
}

func (e *CardGame) PromptAbility(choices []any) {
	e.playableCards = nil
	e.playableFields = nil
	// Re-enable all cards when switching to ability prompt
	e.enableHandCards()
	// Store choices and create menu zones
	e.abilityChoices = choices
	e.abilityMenu = make([]*ui.Zone, len(choices))
	e.abilityMenuHovered = make([]bool, len(choices))

	// Extract the card from the first ability choice
	if len(choices) > 0 {
		if abilityStr, ok := choices[0].(string); ok {
			var cardID int
			if n, _ := fmt.Sscanf(abilityStr, "%d.", &cardID); n == 1 {
				if card, ok := e.cardMap[cardID]; ok {
					e.abilityCard = card
				}
			}
		}
	}

	// Switch focus to ability menu if there are choices (only if keyboard focus is active)
	if len(choices) > 0 && e.focusMode != "" {
		e.focusMode = "ability-menu"
		e.focusAbilityIndex = 0
		// Lower hand when switching to ability menu
		e.hand.hovered = false
		e.LayoutHand()
		// Show attack target preview for initially focused ability
		e.updateAttackTargetPreview()
	}

	for i := range choices {
		choiceIndex := i // Capture loop variable
		e.abilityMenu[i] = &ui.Zone{
			M:       ui.NewImage(30, 3, nil),
			W:       30,
			H:       3,
			Capture: true,
			Enter: func(msg ui.Msg) ui.Cmd {
				e.abilityMenuHovered[choiceIndex] = true

				// Show attack target preview for attack abilities
				if e.abilityCard != nil {
					if abilityStr, ok := e.abilityChoices[choiceIndex].(string); ok {
						var cardID, abilityIdx int
						if n, _ := fmt.Sscanf(abilityStr, "%d.%d", &cardID, &abilityIdx); n == 2 {
							if card, ok := e.cardMap[cardID]; ok && card.cardInstance != nil {
								abilities := card.cardInstance.GetActivatedAbilities()
								if abilityIdx >= 0 && abilityIdx < len(abilities) {
									abilityText := abilities[abilityIdx].Text()
									// Check if this is an attack ability
									if strings.Contains(strings.ToLower(abilityText), "attack") {
										// Find which lane the card is in
										fieldIndex := -1
										lanes := e.playerLanes.(*Lanes)
										for i, c := range lanes.cards {
											if c != nil && c.(*Card) == card {
												fieldIndex = i
												break
											}
										}
										if fieldIndex != -1 {
											e.attackTargetField = fieldIndex
											e.attackTargetTime = time.Now()
											e.attackTargetIsPreview = true
										}
									}
								}
							}
						}
					}
				}
				return nil
			},
			Leave: func(msg ui.Msg) ui.Cmd {
				e.abilityMenuHovered[choiceIndex] = false
				// Clear attack target preview when leaving ability hover
				if e.attackTargetIsPreview {
					e.attackTargetField = -1
					e.attackTargetIsPreview = false
				}
				return nil
			},
			Click: func(msg ui.Msg) ui.Cmd {
				if e.promptingAbility && e.player != nil {
					e.prompting = false
					e.promptingAbility = false
					e.abilityMenu = []*ui.Zone{}
					e.abilityMenuHovered = []bool{}
					e.abilityChoices = []any{}
					e.abilityCard = nil
					// Clear attack target preview
					e.attackTargetField = -1
					e.attackTargetIsPreview = false
					e.enableHandCards()
					e.player.Send(engine.Msg{Selected: []int{choiceIndex}})
				}
				return nil
			},
		}
	}
}

func (e *CardGame) PromptTarget(choices []any) {
	e.targetChoices = choices
	e.targetableCards = nil
	e.targetableFields = nil

	// Parse choices to identify what can be targeted
	for _, choice := range choices {
		switch v := choice.(type) {
		case *engine.CardInstance:
			// This is a card target
			if card, exists := e.cardMap[v.GetId()]; exists {
				e.targetableCards = append(e.targetableCards, card)
			}
		case *engine.Player:
			// This is a player target - we'll use field index -1 for player
			// Determine which player this is
			if v == e.player {
				e.targetableFields = append(e.targetableFields, -1) // -1 = player
			} else if v == e.enemy {
				e.targetableFields = append(e.targetableFields, -2) // -2 = enemy player
			}
		case int:
			// This might be a field index or other integer value
			// Add to targetable fields
			e.targetableFields = append(e.targetableFields, v)
		}
	}

	// Disable all cards on the board (similar to PromptCard)
	if len(e.targetableCards) > 0 {
		e.disableAllCards()

		// Enable only the targetable cards
		for _, targetCard := range e.targetableCards {
			targetCard.Disabled = false
		}
	}

	// Switch focus to appropriate mode (only if keyboard focus is active)
	if e.focusMode != "" {
		// If there are targetable cards on the board, focus on them
		if len(e.targetableCards) > 0 {
			// Check if any targetable cards are in player lanes
			playerLanes := e.playerLanes.(*Lanes)
			enemyLanes := e.enemyLanes.(*Lanes)

			hasPlayerTargets := false
			hasEnemyTargets := false

			for _, targetCard := range e.targetableCards {
				for i, card := range playerLanes.cards {
					if card != nil && card.(*Card) == targetCard {
						hasPlayerTargets = true
						if !hasEnemyTargets {
							e.focusFieldIndex = i
						}
						break
					}
				}
				for i, card := range enemyLanes.cards {
					if card != nil && card.(*Card) == targetCard {
						hasEnemyTargets = true
						e.focusFieldIndex = i
						break
					}
				}
			}

			// Focus on enemy field first if there are enemy targets
			if hasEnemyTargets {
				e.focusMode = "enemy-field"
			} else if hasPlayerTargets {
				e.focusMode = "field"
			}
		} else if len(e.targetableFields) > 0 {
			// Focus on fields/players
			// Check if targeting players (negative indices)
			hasEnemyPlayer := false
			for _, idx := range e.targetableFields {
				if idx == -2 {
					hasEnemyPlayer = true
					break
				}
			}
			if hasEnemyPlayer {
				e.focusMode = "enemy-field"
				e.focusFieldIndex = -2 // Special index for enemy player
			} else {
				e.focusMode = "field"
				e.focusFieldIndex = e.targetableFields[0]
			}
		}
		// Lower hand when switching to target mode
		e.hand.hovered = false
		e.LayoutHand()
	}
}

func (e *CardGame) eventHandler(event *engine.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	player := event.Player

	// Handle card reveal events
	if event.Event == engine.EventOnDraw || event.Event == engine.EventOnEnterBoard {
		// TODO: handle card reveal events
	}
	fmt.Println(event)
	switch event.Event {
	case engine.EventOnDraw:
		if player == e.player {
			e.Draw(event.Args[0].(*engine.CardInstance))
		} else {
			// Draw card for enemy
			cardInstance := event.Args[0].(*engine.CardInstance)
			if _, exists := e.cardMap[cardInstance.GetId()]; !exists {
				e.CreateCard(cardInstance)
			}
		}
	case engine.EventOnEnterBoard:
		// Move card from stack to actual field position
		if len(event.Args) >= 2 {
			cardInstance := event.Args[0].(*engine.CardInstance)
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

				// Re-enable card when leaving stack
				card.Disabled = false

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
	case engine.EventAtStartPhase:
		e.currentPhase = "Start"
		e.updateCurrentPlayer(player)
	case engine.EventAtDrawPhase:
		e.currentPhase = "Draw"
		e.updateCurrentPlayer(player)
	case engine.EventAtPlayPhase:
		e.currentPhase = "Play"
		e.updateCurrentPlayer(player)
	case engine.EventAtEndPhase:
		e.currentPhase = "End"
		e.updateCurrentPlayer(player)
	case engine.EventOnGainLife:
		if len(event.Args) > 0 {
			amount := event.Args[0].(int)
			if player == e.player {
				e.playerLife += amount
			} else {
				e.enemyLife += amount
			}
		}
	case engine.EventOnLoseLife:
		if len(event.Args) > 0 {
			amount := event.Args[0].(int)
			if player == e.player {
				e.playerLife -= amount
				e.playerHealthFlash = time.Now()
			} else {
				e.enemyLife -= amount
				e.enemyHealthFlash = time.Now()
			}
			// Trigger screen shake effect
			e.TriggerScreenShake()
		}
	case engine.EventOnAddEssence:
		if len(event.Args) > 0 {
			essenceType := event.Args[0].(string)
			if player == e.player {
				e.playerEssence[essenceType]++
			} else {
				e.enemyEssence[essenceType]++
			}
		}
	case engine.EventOnRemoveEssence:
		if len(event.Args) > 0 {
			essenceType := event.Args[0].(string)
			if player == e.player {
				if e.playerEssence[essenceType] > 0 {
					e.playerEssence[essenceType]--
				}
			} else {
				if e.enemyEssence[essenceType] > 0 {
					e.enemyEssence[essenceType]--
				}
			}
		}
	case engine.EventOnAttack:
		// Trigger attack bump animation for the attacking card
		if len(event.Args) > 0 {
			if cardInst, ok := event.Args[0].(*engine.CardInstance); ok {
				if card, ok := e.cardMap[cardInst.GetId()]; ok {
					// Determine bump direction based on which player owns the card
					// Player cards (bottom) attack upward (-1), enemy cards (top) attack downward (1)
					direction := 1
					if player == e.player {
						direction = -1
					}
					card.AnimateBump(direction)

					// Show attack target field - find which lane the card is in
					fieldIndex := -1
					var lanes *Lanes
					if player == e.player {
						lanes = e.playerLanes.(*Lanes)
					} else {
						lanes = e.enemyLanes.(*Lanes)
					}
					for i, c := range lanes.cards {
						if c != nil && c.(*Card) == card {
							fieldIndex = i
							break
						}
					}
					if fieldIndex != -1 {
						e.attackTargetField = fieldIndex
						e.attackTargetTime = time.Now()
						e.attackTargetIsPreview = false // This is the actual attack, not preview
					}
				}
			}
		}
	case engine.EventOnDestroy:
		// Remove destroyed card from the board and move to pile
		if len(event.Args) > 0 {
			if cardInst, ok := event.Args[0].(*engine.CardInstance); ok {
				if card, ok := e.cardMap[cardInst.GetId()]; ok {
					// Determine pile position based on player
					var discardX, discardY int
					if player == e.player {
						// Player pile at bottom-left
						discardX = 2
						discardY = e.H - 15
					} else {
						// Enemy pile at top-left
						discardX = 2
						discardY = 2
					}

					// Animate card flying to discard pile
					card.AnimateFlyTo(discardX, discardY)

					// Remove from lanes and add to pile after animation completes
					time.AfterFunc(300*time.Millisecond, func() {
						// Remove from lanes
						if card.Location == CardLocBoard {
							// Determine which lanes to use
							var lanes *Lanes
							if player == e.player {
								lanes = e.playerLanes.(*Lanes)
							} else {
								lanes = e.enemyLanes.(*Lanes)
							}

							// Remove card from lane
							for i, c := range lanes.cards {
								if c != nil && c.(*Card) == card {
									lanes.cards[i] = nil
									break
								}
							}
						}

						// Add to pile collection
						if player == e.player {
							e.playerPile = append(e.playerPile, card)
						} else {
							e.enemyPile = append(e.enemyPile, card)
						}
						// Keep in cardMap for rendering but mark it's in pile
						card.Location = CardLocBoard // Reuse this to mark as "not interactive"
					})
				}
			}
		}
	case engine.EventPromptCard:
		if player == e.player {
			e.prompting = true
			e.PromptCard(event.Args[1:])
		} else {
			e.enemyBotPromptCard(event.Args[1:], player)
		}
	case engine.EventPromptField:
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
	case engine.EventPromptAbility:
		if player == e.player {
			e.prompting = true
			e.promptingAbility = true
			e.PromptAbility(event.Args[1:])
		} else {
			e.enemyBotPromptAbility(event.Args[1:])
		}
	case engine.EventPromptTarget:
		if player == e.player {
			e.prompting = true
			e.promptingTarget = true
			e.enableHandCards()
			e.PromptTarget(event.Args[1:])
		} else {
			e.enemyBotPromptTarget(event.Args[1:])
		}
	case engine.EventPromptSource:
		if player == e.player {
			e.prompting = true
			// Re-enable all cards when switching to source prompt
			e.enableHandCards()
		} else {
			e.enemyBotPromptSource(event.Args[1:])
		}
	case engine.EventPromptDiscard:
		if player == e.player {
			e.prompting = true
			// Re-enable all cards when switching to discard prompt
			e.enableHandCards()
		} else {
			e.enemyBotPromptDiscard(event.Args[1:])
		}
	}
}

func (e *CardGame) updateCurrentPlayer(player *engine.Player) {
	if player == e.player {
		e.currentPlayer = "Player"
	} else if player == e.enemy {
		e.currentPlayer = "Enemy"
	} else {
		e.currentPlayer = "Unknown"
	}
}

// iterateAllCards calls the provided function for each card in hand and on the board
func (e *CardGame) iterateAllCards(fn func(*Card)) {
	// Hand cards
	for _, card := range e.hand.Cards() {
		fn(card.(*Card))
	}
	// Player lane cards
	playerLanes := e.playerLanes.(*Lanes)
	for _, card := range playerLanes.cards {
		if card != nil {
			fn(card.(*Card))
		}
	}
	// Enemy lane cards
	enemyLanes := e.enemyLanes.(*Lanes)
	for _, card := range enemyLanes.cards {
		if card != nil {
			fn(card.(*Card))
		}
	}
}

// iterateHandCards calls the provided function for each card in hand
func (e *CardGame) iterateHandCards(fn func(*Card)) {
	for _, card := range e.hand.Cards() {
		fn(card.(*Card))
	}
}

// iterateLaneCards calls the provided function for each card on the board
func (e *CardGame) iterateLaneCards(fn func(*Card)) {
	playerLanes := e.playerLanes.(*Lanes)
	for _, card := range playerLanes.cards {
		if card != nil {
			fn(card.(*Card))
		}
	}
	enemyLanes := e.enemyLanes.(*Lanes)
	for _, card := range enemyLanes.cards {
		if card != nil {
			fn(card.(*Card))
		}
	}
}

// disableAllCards disables all cards in hand and on the board
func (e *CardGame) disableAllCards() {
	e.iterateAllCards(func(c *Card) { c.Disabled = true })
}

// enableAllCards enables all cards in hand and on the board
func (e *CardGame) enableAllCards() {
	e.iterateAllCards(func(c *Card) { c.Disabled = false })
}

// disableHandCards disables all cards in hand
func (e *CardGame) disableHandCards() {
	e.iterateHandCards(func(c *Card) { c.Disabled = true })
}

// enableHandCards enables all cards in hand
func (e *CardGame) enableHandCards() {
	e.iterateHandCards(func(c *Card) { c.Disabled = false })
}

// disableLaneCards disables all cards on the board
func (e *CardGame) disableLaneCards() {
	e.iterateLaneCards(func(c *Card) { c.Disabled = true })
}

// enableLaneCards enables all cards on the board
func (e *CardGame) enableLaneCards() {
	e.iterateLaneCards(func(c *Card) { c.Disabled = false })
}

// clearAllFocus clears focus from all cards
func (e *CardGame) clearAllFocus() {
	e.iterateAllCards(func(c *Card) { c.Focused = false })
}

// TriggerScreenShake starts a screen shake animation
func (e *CardGame) TriggerScreenShake() {
	// Create a rapid shake sequence with random offsets
	duration := float32(0.05) // 50ms per shake
	shakeAmount := 4          // pixels

	e.shakeSequence = tween.NewSequence(
		tween.New(0, float32(-shakeAmount), duration, tween.OutQuad),
		tween.New(float32(-shakeAmount), float32(shakeAmount), duration, tween.InOutQuad),
		tween.New(float32(shakeAmount), float32(-shakeAmount/2), duration, tween.InOutQuad),
		tween.New(float32(-shakeAmount/2), float32(shakeAmount/2), duration, tween.InOutQuad),
		tween.New(float32(shakeAmount/2), 0, duration, tween.InQuad),
	)
	e.shaking = true
}

// LayoutHand delegates to the hand component
func (e *CardGame) LayoutHand() {
	e.hand.Layout()
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

func (e *CardGame) RemoveCard(card *Card) bool {
	// Remove from hand
	if card.Location == CardLocHand {
		return e.hand.Remove(card)
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
	e.mu.Lock()
	defer e.mu.Unlock()

	var cmds []ui.Cmd

	// Handle screen shake animation and keyboard events
	switch m := msg.(type) {
	case ui.Tick:
		if e.shaking && e.shakeSequence != nil {
			xVal, _, complete := e.shakeSequence.Update(m.DeltaTime)
			e.shakeX = int(xVal)
			if complete {
				e.shaking = false
				e.shakeSequence = nil
				e.shakeX = 0
				e.shakeY = 0
			}
		}
	case ui.KeyEvent:
		if cmd := e.handleKeyPress(m); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

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

	return e, ui.Batch(cmds...)
}

func (e *CardGame) handleKeyPress(key ui.KeyEvent) ui.Cmd {
	// Only handle key press events, not releases
	if !key.Pressed {
		return nil
	}

	// Handle arrow keys and enter for navigation
	switch key.Key {
	case ebiten.KeyRight:
		return e.handleRightArrow()
	case ebiten.KeyLeft:
		return e.handleLeftArrow()
	case ebiten.KeyDown:
		return e.handleDownArrow()
	case ebiten.KeyUp:
		return e.handleUpArrow()
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
		return e.handleEnterKey()
	}
	return nil
}

func (e *CardGame) handleRightArrow() ui.Cmd {
	// Initialize focus on first keyboard use
	if e.focusMode == "" {
		e.focusMode = "hand"
		e.hand.hovered = true
		e.LayoutHand()
		return nil
	}

	if e.focusMode == "ability-menu" {
		// Navigate right in ability menu
		if len(e.abilityMenu) > 0 {
			e.focusAbilityIndex = (e.focusAbilityIndex + 1) % len(e.abilityMenu)
			// Update attack target preview for focused ability
			e.updateAttackTargetPreview()
		}
	} else if e.focusMode == "hand" {
		// Navigate right in hand
		handCards := e.hand.Cards()
		if len(handCards) > 0 {
			// If we have playable cards restriction, cycle through only enabled cards
			if len(e.playableCards) > 0 {
				// Find next enabled card after current position
				startIdx := e.focusHandIndex
				for i := 1; i <= len(handCards); i++ {
					nextIdx := (startIdx + i) % len(handCards)
					if !handCards[nextIdx].(*Card).Disabled {
						e.focusHandIndex = nextIdx
						break
					}
				}
			} else {
				// No restrictions, cycle through all cards
				e.focusHandIndex = (e.focusHandIndex + 1) % len(handCards)
			}
		}
	} else if e.focusMode == "field" {
		// Navigate right through valid fields or targets
		if e.promptingTarget && len(e.targetableCards) > 0 {
			// Navigate through targetable cards in player lanes
			playerLanes := e.playerLanes.(*Lanes)
			validIndices := []int{}
			for i, card := range playerLanes.cards {
				if card != nil {
					for _, targetCard := range e.targetableCards {
						if card.(*Card) == targetCard {
							validIndices = append(validIndices, i)
							break
						}
					}
				}
			}
			if len(validIndices) > 0 {
				currentIdx := -1
				for i, idx := range validIndices {
					if idx == e.focusFieldIndex {
						currentIdx = i
						break
					}
				}
				nextIdx := (currentIdx + 1) % len(validIndices)
				e.focusFieldIndex = validIndices[nextIdx]
			}
		} else if len(e.playableFields) > 0 {
			// Find current index in playableFields
			currentIdx := -1
			for i, fieldIdx := range e.playableFields {
				if fieldIdx == e.focusFieldIndex {
					currentIdx = i
					break
				}
			}
			// Move to next valid field
			nextIdx := (currentIdx + 1) % len(e.playableFields)
			e.focusFieldIndex = e.playableFields[nextIdx]
		} else {
			// No restrictions, cycle through all 5 fields
			e.focusFieldIndex = (e.focusFieldIndex + 1) % 5
		}
	} else if e.focusMode == "enemy-field" {
		// Navigate right in enemy fields or targets
		if e.promptingTarget && len(e.targetableCards) > 0 {
			// Navigate through targetable cards in enemy lanes
			enemyLanes := e.enemyLanes.(*Lanes)
			validIndices := []int{}
			for i, card := range enemyLanes.cards {
				if card != nil {
					for _, targetCard := range e.targetableCards {
						if card.(*Card) == targetCard {
							validIndices = append(validIndices, i)
							break
						}
					}
				}
			}
			if len(validIndices) > 0 {
				currentIdx := -1
				for i, idx := range validIndices {
					if idx == e.focusFieldIndex {
						currentIdx = i
						break
					}
				}
				nextIdx := (currentIdx + 1) % len(validIndices)
				e.focusFieldIndex = validIndices[nextIdx]
			}
		} else {
			// Navigate right in enemy fields (all 5)
			e.focusFieldIndex = (e.focusFieldIndex + 1) % 5
		}
	}
	return nil
}

func (e *CardGame) handleLeftArrow() ui.Cmd {
	// Initialize focus on first keyboard use
	if e.focusMode == "" {
		e.focusMode = "hand"
		e.hand.hovered = true
		e.LayoutHand()
		return nil
	}

	if e.focusMode == "ability-menu" {
		// Navigate left in ability menu
		if len(e.abilityMenu) > 0 {
			e.focusAbilityIndex--
			if e.focusAbilityIndex < 0 {
				e.focusAbilityIndex = len(e.abilityMenu) - 1
			}
			// Update attack target preview for focused ability
			e.updateAttackTargetPreview()
		}
	} else if e.focusMode == "hand" {
		// Navigate left in hand
		handCards := e.hand.Cards()
		if len(handCards) > 0 {
			// If we have playable cards restriction, cycle through only enabled cards
			if len(e.playableCards) > 0 {
				// Find previous enabled card before current position
				startIdx := e.focusHandIndex
				for i := 1; i <= len(handCards); i++ {
					prevIdx := (startIdx - i + len(handCards)) % len(handCards)
					if !handCards[prevIdx].(*Card).Disabled {
						e.focusHandIndex = prevIdx
						break
					}
				}
			} else {
				// No restrictions, cycle through all cards
				e.focusHandIndex--
				if e.focusHandIndex < 0 {
					e.focusHandIndex = len(handCards) - 1
				}
			}
		}
	} else if e.focusMode == "field" {
		// Navigate left through valid fields or targets
		if e.promptingTarget && len(e.targetableCards) > 0 {
			// Navigate through targetable cards in player lanes
			playerLanes := e.playerLanes.(*Lanes)
			validIndices := []int{}
			for i, card := range playerLanes.cards {
				if card != nil {
					for _, targetCard := range e.targetableCards {
						if card.(*Card) == targetCard {
							validIndices = append(validIndices, i)
							break
						}
					}
				}
			}
			if len(validIndices) > 0 {
				currentIdx := -1
				for i, idx := range validIndices {
					if idx == e.focusFieldIndex {
						currentIdx = i
						break
					}
				}
				prevIdx := currentIdx - 1
				if prevIdx < 0 {
					prevIdx = len(validIndices) - 1
				}
				e.focusFieldIndex = validIndices[prevIdx]
			}
		} else if len(e.playableFields) > 0 {
			// Find current index in playableFields
			currentIdx := -1
			for i, fieldIdx := range e.playableFields {
				if fieldIdx == e.focusFieldIndex {
					currentIdx = i
					break
				}
			}
			// Move to previous valid field
			prevIdx := currentIdx - 1
			if prevIdx < 0 {
				prevIdx = len(e.playableFields) - 1
			}
			e.focusFieldIndex = e.playableFields[prevIdx]
		} else {
			// No restrictions, cycle through all 5 fields
			e.focusFieldIndex--
			if e.focusFieldIndex < 0 {
				e.focusFieldIndex = 4
			}
		}
	} else if e.focusMode == "enemy-field" {
		// Navigate left in enemy fields or targets
		if e.promptingTarget && len(e.targetableCards) > 0 {
			// Navigate through targetable cards in enemy lanes
			enemyLanes := e.enemyLanes.(*Lanes)
			validIndices := []int{}
			for i, card := range enemyLanes.cards {
				if card != nil {
					for _, targetCard := range e.targetableCards {
						if card.(*Card) == targetCard {
							validIndices = append(validIndices, i)
							break
						}
					}
				}
			}
			if len(validIndices) > 0 {
				currentIdx := -1
				for i, idx := range validIndices {
					if idx == e.focusFieldIndex {
						currentIdx = i
						break
					}
				}
				prevIdx := currentIdx - 1
				if prevIdx < 0 {
					prevIdx = len(validIndices) - 1
				}
				e.focusFieldIndex = validIndices[prevIdx]
			}
		} else {
			// Navigate left in enemy fields (all 5)
			e.focusFieldIndex--
			if e.focusFieldIndex < 0 {
				e.focusFieldIndex = 4
			}
		}
	}
	return nil
}

func (e *CardGame) handleDownArrow() ui.Cmd {
	// Initialize focus on first keyboard use
	if e.focusMode == "" {
		e.focusMode = "hand"
		e.hand.hovered = true
		e.LayoutHand()
		return nil
	}

	// Cycle through modes: hand -> enemy-field -> field -> hand
	if e.focusMode == "hand" {
		e.focusMode = "enemy-field"
		e.focusFieldIndex = 0
		// Lower hand when losing focus
		e.hand.hovered = false
		e.LayoutHand()
	} else if e.focusMode == "enemy-field" {
		e.focusMode = "field"
		e.focusFieldIndex = 0
	} else if e.focusMode == "field" {
		e.focusMode = "hand"
		e.focusHandIndex = 0
		// Raise hand when gaining focus
		e.hand.hovered = true
		e.LayoutHand()
	}
	return nil
}

func (e *CardGame) handleUpArrow() ui.Cmd {
	// Initialize focus on first keyboard use
	if e.focusMode == "" {
		e.focusMode = "hand"
		e.hand.hovered = true
		e.LayoutHand()
		return nil
	}

	// Cycle through modes in reverse: hand -> field -> enemy-field -> hand
	if e.focusMode == "hand" {
		e.focusMode = "field"
		e.focusFieldIndex = 0
		// Lower hand when losing focus
		e.hand.hovered = false
		e.LayoutHand()
	} else if e.focusMode == "field" {
		e.focusMode = "enemy-field"
		e.focusFieldIndex = 0
	} else if e.focusMode == "enemy-field" {
		e.focusMode = "hand"
		e.focusHandIndex = 0
		// Raise hand when gaining focus
		e.hand.hovered = true
		e.LayoutHand()
	}
	return nil
}

func (e *CardGame) handleEnterKey() ui.Cmd {
	// Initialize focus on first keyboard use
	if e.focusMode == "" {
		e.focusMode = "hand"
		e.hand.hovered = true
		e.LayoutHand()
		return nil
	}

	if !e.prompting || e.player == nil {
		return nil
	}

	// Handle ability menu selection
	if e.promptingAbility && len(e.abilityMenu) > 0 && e.focusMode == "ability-menu" {
		// Select the focused ability
		if e.focusAbilityIndex >= 0 && e.focusAbilityIndex < len(e.abilityChoices) {
			e.player.Send(engine.Msg{Selected: []int{e.focusAbilityIndex}})
			e.promptingAbility = false
			e.abilityMenu = []*ui.Zone{}
			e.abilityMenuHovered = []bool{}
			e.abilityChoices = []any{}
			e.abilityCard = nil
			// Clear attack target preview
			e.attackTargetField = -1
			e.attackTargetIsPreview = false
			return nil
		}
		return e.handleSkipAction()
	}

	// Handle target selection
	if e.promptingTarget && len(e.targetChoices) > 0 {
		// Check if targeting cards on fields
		if e.focusMode == "field" || e.focusMode == "enemy-field" {
			var lanes *Lanes
			if e.focusMode == "field" {
				lanes = e.playerLanes.(*Lanes)
			} else {
				lanes = e.enemyLanes.(*Lanes)
			}

			// Check if targeting a card
			if e.focusFieldIndex >= 0 && e.focusFieldIndex < len(lanes.cards) {
				card := lanes.cards[e.focusFieldIndex]
				if card != nil {
					cardObj := card.(*Card)
					// Check if this card is targetable
					for i, choice := range e.targetChoices {
						if cardInst, ok := choice.(*engine.CardInstance); ok {
							if cardObj.cardInstance != nil && cardInst.GetId() == cardObj.cardInstance.GetId() {
								// Found valid target - send selection
								e.player.Send(engine.Msg{Selected: []int{i}})
								e.promptingTarget = false
								e.targetChoices = nil
								e.targetableCards = nil
								e.targetableFields = nil
								// Re-enable all cards on board
								e.enableLaneCards()
								return nil
							}
						}
					}
				}
			} else if e.focusFieldIndex == -1 || e.focusFieldIndex == -2 {
				// Targeting a player
				for i, choice := range e.targetChoices {
					if playerChoice, ok := choice.(*engine.Player); ok {
						if (e.focusFieldIndex == -1 && playerChoice == e.player) ||
							(e.focusFieldIndex == -2 && playerChoice == e.enemy) {
							// Found valid player target - send selection
							e.player.Send(engine.Msg{Selected: []int{i}})
							e.promptingTarget = false
							e.targetChoices = nil
							e.targetableCards = nil
							e.targetableFields = nil
							// Re-enable all cards on board
							e.enableLaneCards()
							return nil
						}
					}
				}
			}
		}
		// Target not valid, allow skip
		return e.handleSkipAction()
	}

	// Handle card selection from hand (playing a card)
	if len(e.playableCards) > 0 && e.focusMode == "hand" {
		handCards := e.hand.Cards()
		if e.focusHandIndex >= 0 && e.focusHandIndex < len(handCards) {
			card := handCards[e.focusHandIndex].(*Card)
			if !card.Disabled && card.cardInstance != nil {
				// Check if this card is in the playable cards list
				for i, playableCard := range e.playableCards {
					if playableCard == card {
						// Set as selected card so it can be put on stack later
						e.selectedCard = card
						// Send card selection to game
						e.player.Send(engine.Msg{Selected: []int{i}})
						return nil
					}
				}
			}
		}
		// If we're prompting for cards but didn't select one, allow skip
		return e.handleSkipAction()
	}

	// Handle field selection (placing a card)
	if len(e.playableFields) > 0 && e.focusMode == "field" {
		// Find the index in playableFields that matches focusFieldIndex
		for i, fieldIdx := range e.playableFields {
			if fieldIdx == e.focusFieldIndex {
				// If we have a selected card, put it on the stack first
				if e.selectedCard != nil {
					// Remove from hand
					wasInHand := e.selectedCard.Location == CardLocHand
					e.RemoveCard(e.selectedCard)

					// Put card on the stack zone
					e.stack.SetCard(e.selectedCard, fieldIdx)

					// Card is no longer in hand, disable hover
					e.selectedCard.hovered = false
					e.selectedCard.hoverOffset = 0

					// Rearrange remaining cards in hand if this card came from hand
					if wasInHand {
						e.LayoutHand()
					}
				}

				// Send field selection to game
				e.player.Send(engine.Msg{Selected: []int{i}})
				e.prompting = false
				e.playableFields = nil
				e.validFields = nil
				e.selectedCard = nil
				return nil
			}
		}
		// Field not in valid list, allow skip
		return e.handleSkipAction()
	}

	// Handle card selection on field (activating abilities)
	// Only allow this when prompting for cards and not placing a card
	if e.prompting && len(e.playableFields) == 0 && e.focusMode == "field" && len(e.cardChoices) > 0 {
		playerLanes := e.playerLanes.(*Lanes)
		if e.focusFieldIndex >= 0 && e.focusFieldIndex < len(playerLanes.cards) {
			card := playerLanes.cards[e.focusFieldIndex]
			if card != nil {
				cardObj := card.(*Card)
				if cardObj.cardInstance != nil {
					// Find this card in the stored choices and send its index
					if e.player != nil {
						// Find the index of this card in cardChoices array
						for i, choice := range e.cardChoices {
							if choiceCard, ok := choice.(*engine.CardInstance); ok {
								if choiceCard.GetId() == cardObj.cardInstance.GetId() {
									// Found it! Send the actual index in the choices array
									e.player.Send(engine.Msg{Selected: []int{i}})
									return nil
								}
							}
						}
					}
				}
			}
		}
		// No matching card found or empty field, allow skip
		return e.handleSkipAction()
	}

	// Handle enemy field selection (attack target)
	if e.focusMode == "enemy-field" {
		// For now, treat as skip - would need to track attack targets
		return e.handleSkipAction()
	}

	// Default: skip action
	return e.handleSkipAction()
}

func (e *CardGame) updateAttackTargetPreview() {
	// Clear any existing preview first
	if e.attackTargetIsPreview {
		e.attackTargetField = -1
		e.attackTargetIsPreview = false
	}

	// Check if the focused ability is an attack ability
	if e.focusMode == "ability-menu" && e.focusAbilityIndex >= 0 && e.focusAbilityIndex < len(e.abilityChoices) && e.abilityCard != nil {
		if abilityStr, ok := e.abilityChoices[e.focusAbilityIndex].(string); ok {
			var cardID, abilityIdx int
			if n, _ := fmt.Sscanf(abilityStr, "%d.%d", &cardID, &abilityIdx); n == 2 {
				if card, ok := e.cardMap[cardID]; ok && card.cardInstance != nil {
					abilities := card.cardInstance.GetActivatedAbilities()
					if abilityIdx >= 0 && abilityIdx < len(abilities) {
						abilityText := abilities[abilityIdx].Text()
						// Check if this is an attack ability
						if strings.Contains(strings.ToLower(abilityText), "attack") {
							// Find which lane the card is in
							fieldIndex := -1
							lanes := e.playerLanes.(*Lanes)
							for i, c := range lanes.cards {
								if c != nil && c.(*Card) == card {
									fieldIndex = i
									break
								}
							}
							if fieldIndex != -1 {
								e.attackTargetField = fieldIndex
								e.attackTargetTime = time.Now()
								e.attackTargetIsPreview = true
							}
						}
					}
				}
			}
		}
	}
}

func (e *CardGame) handleSkipAction() ui.Cmd {
	if e.player == nil {
		return nil
	}

	e.player.Send(engine.Msg{Selected: []int{engine.SkipCode}})
	e.prompting = false
	e.promptingAbility = false
	e.promptingTarget = false
	e.playableCards = nil
	e.playableFields = nil
	e.cardChoices = nil
	e.validFields = nil
	e.abilityMenu = []*ui.Zone{}
	e.abilityMenuHovered = []bool{}
	e.abilityChoices = []any{}
	e.abilityCard = nil
	e.targetChoices = nil
	e.targetableCards = nil
	e.targetableFields = nil
	e.selectedCard = nil
	e.attackTargetField = -1
	e.attackTargetIsPreview = false

	// Re-enable all cards
	e.enableAllCards()

	return nil
}

func (e *CardGame) updateFocusIndicators() {
	// Clear all focus indicators first
	e.clearAllFocus()

	playerLanes := e.playerLanes.(*Lanes)
	enemyLanes := e.enemyLanes.(*Lanes)

	// Set focus on the appropriate card/field
	if e.focusMode == "hand" {
		handCards := e.hand.Cards()
		if e.focusHandIndex >= 0 && e.focusHandIndex < len(handCards) {
			handCards[e.focusHandIndex].(*Card).Focused = true
		}
	} else if e.focusMode == "field" {
		if e.focusFieldIndex >= 0 && e.focusFieldIndex < len(playerLanes.cards) {
			card := playerLanes.cards[e.focusFieldIndex]
			if card != nil {
				card.(*Card).Focused = true
			}
		}
	} else if e.focusMode == "enemy-field" {
		if e.focusFieldIndex >= 0 && e.focusFieldIndex < len(enemyLanes.cards) {
			card := enemyLanes.cards[e.focusFieldIndex]
			if card != nil {
				card.(*Card).Focused = true
			}
		}
	}
}

func (e *CardGame) View() *ui.Image {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Update focus indicators
	e.updateFocusIndicators()

	screen := ui.NewImage(e.W, e.H, nil)

	// Draw hand zone (positioned to cover raised cards)
	screen = screen.Overlay(e.hand.View(), e.shakeX, e.H-10+e.shakeY)

	// Draw enemy lanes at the top with resource zones
	screen = screen.Overlay(ui.JoinVertical(
		ui.Center,
		renderZoneWithHover(e.enemyResources.Zone(), e.enemyResources.IsHovered(), ui.Borders["round"], ui.Colors["dark-brown"], ui.Colors["brown"]),
		e.enemyLanes.View(),
		e.playerLanes.View(),
		renderZoneWithHover(e.playerResources.Zone(), e.playerResources.IsHovered(), ui.Borders["round"], ui.Colors["dark-brown"], ui.Colors["brown"]),
	), e.shakeX, e.shakeY)

	// Render UI components (all with shake offset)
	screen = e.renderEssencePools(screen)
	screen = e.renderLifeTotals(screen)
	screen = e.renderCards(screen)
	screen = e.renderStack(screen)
	screen = e.renderPhaseLabel(screen)
	screen = e.renderAbilityMenu(screen)
	screen = e.renderSkipButton(screen)

	return screen
}

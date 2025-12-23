package godot

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SvenDH/go-card-engine/engine"

	"graphics.gd/classdb"
	"graphics.gd/classdb/BoxContainer"
	"graphics.gd/classdb/Button"
	"graphics.gd/classdb/Control"
	"graphics.gd/classdb/GUI"
	"graphics.gd/classdb/HBoxContainer"
	"graphics.gd/classdb/Input"
	"graphics.gd/classdb/InputEvent"
	"graphics.gd/classdb/InputEventMouseButton"
	"graphics.gd/classdb/InputEventMouseMotion"
	"graphics.gd/classdb/Label"
	"graphics.gd/classdb/MeshInstance3D"
	"graphics.gd/classdb/Node"
	"graphics.gd/classdb/RichTextLabel"
	"graphics.gd/classdb/SubViewport"
	"graphics.gd/classdb/SubViewportContainer"
	"graphics.gd/classdb/Tween"
	"graphics.gd/classdb/VBoxContainer"
	"graphics.gd/variant/Color"
	"graphics.gd/variant/Float"
	"graphics.gd/variant/Object"
	"graphics.gd/variant/Vector2"
	"graphics.gd/variant/Vector2i"
	"graphics.gd/variant/Vector3"
)

const (
	defaultBoardSlots = 5
	startingLife      = 20
)

// CardGameUI is a Godot/graphics.gd driven UI for the card engine.
// It listens for engine events and reflects them with simple controls and prompts.
type CardGameUI struct {
	Control.Extension[CardGameUI]

	statusLabel      Label.Instance
	phaseLabel       Label.Instance
	playerLifeLabel  Label.Instance
	enemyLifeLabel   Label.Instance
	handArea         Control.Instance
	handContainer    SubViewportContainer.Instance
	handViewport     SubViewport.Instance
	playerBoardSlots []Button.Instance
	enemyBoardSlots  []Button.Instance

	promptLabel    Label.Instance
	promptBox      VBoxContainer.Instance
	promptButtons  []Button.Instance
	currentPrompt  engine.EventType
	promptPlayer   *engine.Player
	promptChoices  []any
	promptExpected int

	logLabel RichTextLabel.Instance
	logLines []string

	hand3d  *hand3DScene
	board3d *board3DScene

	eventQueue chan func()

	game   *engine.GameState
	player *engine.Player
	enemy  *engine.Player

	cardViews  map[int]*cardView
	hand       *handScene
	dragging   *cardView
	dragOffset Vector2.XY
	turn       int
}

type cardView struct {
	instance   *engine.CardInstance
	owner      *engine.Player
	button     Button.Instance
	location   string
	fieldIndex int
	homePos    Vector2.XY
	moveTween  Tween.Instance
	hovered    bool
	mesh       MeshInstance3D.Instance
}

func init() {
	classdb.Register[CardGameUI]()
}

// NewCardGameUI creates a new UI controller instance.
func NewCardGameUI() *CardGameUI {
	return &CardGameUI{
		cardViews: make(map[int]*cardView),
	}
}

// Ready sets up the UI layout and kicks off the engine loop.
func (c *CardGameUI) Ready() {
	c.eventQueue = make(chan func(), 256)

	root := c.AsControl()
	root.SetAnchorsPreset(Control.PresetFullRect)

	layout := VBoxContainer.New()
	layout.AsControl().SetAnchorsPreset(Control.PresetFullRect)
	layout.AsControl().SetOffsetsPreset(Control.PresetFullRect)
	layout.AsControl().SetSizeFlagsHorizontal(Control.SizeExpandFill)
	layout.AsControl().SetSizeFlagsVertical(Control.SizeExpandFill)
	layout.AsBoxContainer().SetAlignment(BoxContainer.AlignmentCenter)
	layout.AsControl().SetGrowHorizontal(Control.GrowDirectionBoth)
	layout.AsControl().SetGrowVertical(Control.GrowDirectionBoth)
	root.AsNode().AddChild(layout.AsNode())

	header := HBoxContainer.New()
	header.AsBoxContainer().SetAlignment(BoxContainer.AlignmentCenter)
	layout.AsNode().AddChild(header.AsNode())

	c.statusLabel = Label.New()
	c.statusLabel.SetText("Loading card engine...")
	header.AsNode().AddChild(c.statusLabel.AsNode())

	c.phaseLabel = Label.New()
	c.phaseLabel.SetText("Phase: --")
	header.AsNode().AddChild(c.phaseLabel.AsNode())

	c.playerLifeLabel = Label.New()
	c.playerLifeLabel.SetText("You: ?")
	header.AsNode().AddChild(c.playerLifeLabel.AsNode())

	c.enemyLifeLabel = Label.New()
	c.enemyLifeLabel.SetText("Enemy: ?")
	header.AsNode().AddChild(c.enemyLifeLabel.AsNode())

	boardBox := VBoxContainer.New()
	layout.AsNode().AddChild(boardBox.AsNode())
	boardBox.AsCanvasItem().SetVisible(false)

	enemyLabel := Label.New()
	enemyLabel.SetText("Enemy Board")
	boardBox.AsNode().AddChild(enemyLabel.AsNode())
	enemyRow := HBoxContainer.New()
	enemyRow.AsBoxContainer().SetAlignment(BoxContainer.AlignmentCenter)
	boardBox.AsNode().AddChild(enemyRow.AsNode())
	c.enemyBoardSlots = make([]Button.Instance, defaultBoardSlots)
	for i := 0; i < defaultBoardSlots; i++ {
		btn := Button.New()
		btn.SetText(fmt.Sprintf("Slot %d: empty", i))
		btn.AsBaseButton().SetDisabled(true)
		c.enemyBoardSlots[i] = btn
		enemyRow.AsNode().AddChild(btn.AsNode())
	}

	playerLabel := Label.New()
	playerLabel.SetText("Your Board")
	boardBox.AsNode().AddChild(playerLabel.AsNode())
	playerRow := HBoxContainer.New()
	playerRow.AsBoxContainer().SetAlignment(BoxContainer.AlignmentCenter)
	boardBox.AsNode().AddChild(playerRow.AsNode())
	c.playerBoardSlots = make([]Button.Instance, defaultBoardSlots)
	for i := 0; i < defaultBoardSlots; i++ {
		btn := Button.New()
		btn.SetText(fmt.Sprintf("Slot %d: empty", i))
		btn.AsBaseButton().SetDisabled(true)
		c.playerBoardSlots[i] = btn
		playerRow.AsNode().AddChild(btn.AsNode())
	}

	c.promptLabel = Label.New()
	c.promptLabel.SetText("Waiting for prompts...")
	layout.AsNode().AddChild(c.promptLabel.AsNode())

	c.promptBox = VBoxContainer.New()
	c.promptBox.AsBoxContainer().SetAlignment(BoxContainer.AlignmentCenter)
	layout.AsNode().AddChild(c.promptBox.AsNode())

	c.logLabel = RichTextLabel.New()
	c.logLabel.SetText("Log ready.")
	layout.AsNode().AddChild(c.logLabel.AsNode())

	spacer := Control.New()
	spacer.AsControl().SetSizeFlagsVertical(Control.SizeExpandFill)
	spacer.AsControl().SetSizeFlagsHorizontal(Control.SizeExpandFill)
	layout.AsNode().AddChild(spacer.AsNode())

	handLabel := Label.New()
	handLabel.AsControl().SetSizeFlagsHorizontal(Control.SizeShrinkCenter)
	handLabel.SetHorizontalAlignment(GUI.HorizontalAlignmentCenter)
	c.handContainer = SubViewportContainer.New()
	c.handContainer.SetStretch(true)
	c.handContainer.AsControl().SetCustomMinimumSize(Vector2.XY{720, 240})
	c.handContainer.AsControl().SetMouseFilter(Control.MouseFilterPass)
	c.handContainer.AsControl().SetSizeFlagsHorizontal(Control.SizeExpandFill)
	c.handContainer.AsControl().SetSizeFlagsVertical(Control.SizeShrinkCenter)

	c.handViewport = SubViewport.New()
	c.handViewport.SetSize(Vector2i.New(720, 240))
	c.handViewport.SetSize2dOverrideStretch(true)
	c.handContainer.AsNode().AddChild(c.handViewport.AsNode())
	c.handContainer.AsControl().OnGuiInput(func(ev InputEvent.Instance) {
		if c.hand3d == nil || !c.hand3d.ready() {
			return
		}
		if mm, ok := Object.As[InputEventMouseMotion.Instance](ev); ok {
			gpos := mm.AsInputEventMouse().GlobalPosition()
			if c.dragging != nil {
				if c.dragging.mesh != MeshInstance3D.Nil {
					c.dragging.mesh.AsNode3D().SetPosition(c.mapPointerTo3D(gpos))
				}
				return
			}
			c.hand3d.HoverAt(c.viewportPosition(gpos))
			return
		}
		if mb, ok := Object.As[InputEventMouseButton.Instance](ev); ok {
			if mb.ButtonIndex() != Input.MouseButtonLeft {
				return
			}
			gpos := mb.AsInputEventMouse().GlobalPosition()
			if mb.AsInputEvent().IsPressed() {
				if view := c.hand3d.HoverAt(c.viewportPosition(gpos)); view != nil {
					c.dragging = view
					view.hovered = true
				}
				return
			}
			if c.dragging != nil {
				view := c.dragging
				c.dragging = nil
				view.hovered = false
				c.dropCard3D(view, gpos)
				c.layoutHand()
			}
		}
	})

	c.handArea = c.handContainer.AsControl()
	layout.AsNode().AddChild(c.handContainer.AsNode())
	c.hand3d = newHand3DScene(c, c.handViewport)
	if c.hand3d != nil {
		c.board3d = newBoard3DScene(c.hand3d.rootNode(), defaultBoardSlots, c.hand3d.cardHeight)
	}
	c.hand = newHandScene(c)

	go c.startGameLoop()
}

// Process pumps any pending game events onto the Godot thread each frame.
func (c *CardGameUI) Process(delta Float.X) {
	for i := 0; i < 12; i++ {
		select {
		case fn := <-c.eventQueue:
			fn()
		default:
			return
		}
	}
}

func (c *CardGameUI) startGameLoop() {
	deck, err := loadDeck()
	if err != nil {
		c.queue(func() {
			msg := fmt.Sprintf("Failed to load cards: %v", err)
			c.statusLabel.SetText(msg)
			c.logf("%s", msg)
		})
		return
	}

	rand.Seed(time.Now().UnixNano())

	game := engine.NewGame()
	player := game.AddPlayer(deck...)
	enemy := game.AddPlayer(deck...)

	c.game = game
	c.player = player
	c.enemy = enemy
	c.turn = 1

	c.queue(func() {
		c.statusLabel.SetText("Game running")
		c.playerLifeLabel.SetText(fmt.Sprintf("You: %d", startingLife))
		c.enemyLifeLabel.SetText(fmt.Sprintf("Enemy: %d", startingLife))
		c.phaseLabel.SetText("Phase: start")
	})

	game.On(engine.AllEvents, c.handleEngineEvent)
	game.Run()

	c.queue(func() {
		c.statusLabel.SetText("Game ended")
	})
}

func (c *CardGameUI) handleEngineEvent(event *engine.Event) {
	if c.eventQueue == nil {
		return
	}
	// Copy the event so closures don't race on shared memory.
	ev := *event
	select {
	case c.eventQueue <- func() { c.applyEvent(&ev) }:
	default:
		go func(fn func()) { c.eventQueue <- fn }(func() { c.applyEvent(&ev) })
	}
}

func (c *CardGameUI) applyEvent(event *engine.Event) {
	switch event.Event {
	case engine.EventAtStartPhase:
		c.turn += 1
		c.setPhase("start", event.Player)
	case engine.EventAtDrawPhase:
		c.setPhase("draw", event.Player)
	case engine.EventAtPlayPhase:
		c.setPhase("play", event.Player)
	case engine.EventAtEndPhase:
		c.setPhase("end", event.Player)
	case engine.EventOnDraw:
		if len(event.Args) > 0 {
			if card, ok := event.Args[0].(*engine.CardInstance); ok {
				c.onDrawCard(card, event.Player)
			}
		}
	case engine.EventOnEnterBoard:
		if len(event.Args) >= 2 {
			card, _ := event.Args[0].(*engine.CardInstance)
			if card != nil {
				if idx, ok := event.Args[1].(int); ok {
					c.onEnterBoard(card, event.Player, idx)
				}
			}
		}
	case engine.EventOnLeaveBoard:
		if len(event.Args) > 0 {
			if card, ok := event.Args[0].(*engine.CardInstance); ok {
				c.onLeaveBoard(card)
			}
		}
	case engine.EventOnLoseLife:
		if len(event.Args) > 0 {
			if amount, ok := event.Args[0].(int); ok {
				c.adjustLife(event.Player, -amount)
			}
		}
	case engine.EventOnGainLife:
		if len(event.Args) > 0 {
			if amount, ok := event.Args[0].(int); ok {
				c.adjustLife(event.Player, amount)
			}
		}
	case engine.EventPromptCard:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptField:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptAbility:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptTarget:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptSource:
		c.showPrompt(event.Event, event.Player, event.Args)
	case engine.EventPromptDiscard:
		c.showPrompt(event.Event, event.Player, event.Args)
	}
}

func (c *CardGameUI) onDrawCard(card *engine.CardInstance, owner *engine.Player) {
	view := c.cardViews[card.GetId()]
	if view == nil {
		view = c.createCardView(card, owner)
	}
	view.location = "hand"
	view.fieldIndex = -1
	if owner == c.player {
		view.button.AsBaseButton().SetDisabled(false)
		if c.hand != nil {
			c.hand.Add(view)
		}
	}
	if owner == c.enemy {
		view.button.AsBaseButton().SetDisabled(true)
	}
	c.logf("%s drew %s", c.playerName(owner), card.GetName())
}

func (c *CardGameUI) onEnterBoard(card *engine.CardInstance, owner *engine.Player, index int) {
	view := c.cardViews[card.GetId()]
	if view == nil {
		view = c.createCardView(card, owner)
	}
	if index < 0 {
		return
	}
	view.location = "board"
	view.fieldIndex = index
	view.button.AsBaseButton().SetDisabled(true)
	c.removeFromHand(view)

	slotText := fmt.Sprintf("%s (%s/%s)", card.GetName(), card.GetPower().String(), card.GetHealth().String())
	if owner == c.player && index < len(c.playerBoardSlots) {
		c.playerBoardSlots[index].SetText(slotText)
		c.layoutHand()
	} else if owner == c.enemy && index < len(c.enemyBoardSlots) {
		c.enemyBoardSlots[index].SetText(slotText)
	}
	c.logf("%s placed %s on field %d", c.playerName(owner), card.GetName(), index)
}

func (c *CardGameUI) onLeaveBoard(card *engine.CardInstance) {
	view := c.cardViews[card.GetId()]
	if view == nil || view.fieldIndex < 0 {
		return
	}
	if c.board3d != nil {
		c.board3d.Remove(view)
	}
	index := view.fieldIndex
	if view.owner == c.player && index < len(c.playerBoardSlots) {
		c.playerBoardSlots[index].SetText(fmt.Sprintf("Slot %d: empty", index))
	} else if view.owner == c.enemy && index < len(c.enemyBoardSlots) {
		c.enemyBoardSlots[index].SetText(fmt.Sprintf("Slot %d: empty", index))
	}
	view.fieldIndex = -1
}

func (c *CardGameUI) createCardView(card *engine.CardInstance, owner *engine.Player) *cardView {
	btn := Button.New()
	btn.SetText(fmt.Sprintf("%s (%s/%s)", card.GetName(), card.GetPower().String(), card.GetHealth().String()))
	view := &cardView{
		instance:   card,
		owner:      owner,
		button:     btn,
		location:   "deck",
		fieldIndex: -1,
	}
	btn.AsControl().SetCustomMinimumSize(Vector2.XY{120, 170})
	btn.AsControl().SetMouseFilter(Control.MouseFilterStop)
	btn.AsCanvasItem().SetModulate(Color.RGBA{R: 1, G: 1, B: 1, A: 0})
	btn.AsControl().OnMouseEntered(func() {
		if c.hand3d != nil {
			return
		}
		if c.dragging == view {
			return
		}
		view.hovered = true
		c.layoutHand()
	})
	btn.AsControl().OnMouseExited(func() {
		if c.hand3d != nil {
			return
		}
		if c.dragging == view {
			return
		}
		view.hovered = false
		c.layoutHand()
	})
	btn.AsControl().OnGuiInput(func(ev InputEvent.Instance) {
		// Only log/show card text on click; dragging handled in hand viewport input.
		if mb, ok := Object.As[InputEventMouseButton.Instance](ev); ok {
			if mb.ButtonIndex() == Input.MouseButtonLeft && mb.AsInputEvent().IsPressed() {
				c.logf("%s\n%s", card.GetName(), card.Card.Text)
			}
		}
	})
	btn.AsBaseButton().OnPressed(func() {
		c.logf("%s\n%s", card.GetName(), card.Card.Text)
	})
	c.cardViews[card.GetId()] = view
	return view
}

func (c *CardGameUI) removeFromHand(view *cardView) {
	if c.hand != nil {
		c.hand.Remove(view)
		return
	}
	parent := view.button.AsNode().GetParent()
	if parent != Node.Nil {
		parent.RemoveChild(view.button.AsNode())
	}
	c.layoutHand()
}

func (c *CardGameUI) adjustLife(player *engine.Player, delta int) {
	if player == c.player {
		newLife := extractLife(c.playerLifeLabel.Text()) + delta
		c.playerLifeLabel.SetText(fmt.Sprintf("You: %d", newLife))
	} else if player == c.enemy {
		newLife := extractLife(c.enemyLifeLabel.Text()) + delta
		c.enemyLifeLabel.SetText(fmt.Sprintf("Enemy: %d", newLife))
	}
}

func (c *CardGameUI) setPhase(phase string, player *engine.Player) {
	c.phaseLabel.SetText(fmt.Sprintf("Phase: %s (%s)", phase, c.playerName(player)))
}

func (c *CardGameUI) showPrompt(kind engine.EventType, player *engine.Player, args []any) {
	if player != c.player {
		// Bot answers for the enemy.
		c.botRespond(kind, args)
		return
	}

	c.clearPrompt()

	num := 1
	if len(args) > 0 {
		if v, ok := args[0].(int); ok {
			num = v
		}
	}
	choices := []any{}
	if len(args) > 1 {
		choices = args[1:]
	}

	c.currentPrompt = kind
	c.promptPlayer = player
	c.promptChoices = choices
	c.promptExpected = num

	title := ""
	switch kind {
	case engine.EventPromptCard:
		title = "Select a card to play/activate"
	case engine.EventPromptField:
		title = "Choose a board slot"
	case engine.EventPromptAbility:
		title = "Choose an ability"
	case engine.EventPromptTarget:
		title = "Choose a target"
	case engine.EventPromptSource:
		title = "Use as source?"
	case engine.EventPromptDiscard:
		title = "Choose a card to discard"
	}
	c.promptLabel.SetText(title)

	if kind == engine.EventPromptSource && len(choices) == 0 {
		choices = []any{"Play card", "Use as source"}
	}

	for i, choice := range choices {
		idx := i
		btn := Button.New()
		btn.SetText(fmt.Sprintf("%d: %s", i+1, describeChoice(choice)))
		btn.AsBaseButton().OnPressed(func() {
			c.sendSelection(idx)
		})
		c.promptButtons = append(c.promptButtons, btn)
		c.promptBox.AsNode().AddChild(btn.AsNode())
	}

	skip := Button.New()
	skip.SetText("Skip")
	skip.AsBaseButton().OnPressed(func() {
		c.sendSelection(engine.SkipCode)
	})
	c.promptButtons = append(c.promptButtons, skip)
	c.promptBox.AsNode().AddChild(skip.AsNode())
}

func (c *CardGameUI) sendSelection(value int) {
	if c.promptPlayer != nil {
		c.promptPlayer.Send(engine.Msg{Selected: []int{value}})
	}
	c.clearPrompt()
}

func (c *CardGameUI) clearPrompt() {
	c.promptLabel.SetText("Waiting for prompts...")
	for _, btn := range c.promptButtons {
		parent := btn.AsNode().GetParent()
		if parent != Node.Nil {
			parent.RemoveChild(btn.AsNode())
		}
		btn.AsNode().QueueFree()
	}
	c.promptButtons = nil
	c.currentPrompt = engine.NoEvent
	c.promptPlayer = nil
	c.promptChoices = nil
	c.promptExpected = 0
}

func (c *CardGameUI) botRespond(kind engine.EventType, args []any) {
	switch kind {
	case engine.EventPromptCard:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		idx := rand.Intn(len(choices))
		c.enemy.Send(engine.Msg{Selected: []int{idx}})
	case engine.EventPromptField:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		c.enemy.Send(engine.Msg{Selected: []int{rand.Intn(len(choices))}})
	case engine.EventPromptAbility:
		c.enemy.Send(engine.Msg{Selected: []int{0}})
	case engine.EventPromptTarget:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		c.enemy.Send(engine.Msg{Selected: []int{rand.Intn(len(choices))}})
	case engine.EventPromptSource:
		// 80% chance to use as source.
		if rand.Float32() < 0.8 {
			c.enemy.Send(engine.Msg{Selected: []int{1}})
		} else {
			c.enemy.Send(engine.Msg{Selected: []int{0}})
		}
	case engine.EventPromptDiscard:
		choices := args[1:]
		if len(choices) == 0 {
			c.enemy.Send(engine.Msg{Selected: []int{engine.SkipCode}})
			return
		}
		c.enemy.Send(engine.Msg{Selected: []int{rand.Intn(len(choices))}})
	}
}

func (c *CardGameUI) playerName(p *engine.Player) string {
	switch p {
	case c.player:
		return "You"
	case c.enemy:
		return "Enemy"
	default:
		return "Unknown"
	}
}

func (c *CardGameUI) logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	c.logLines = append(c.logLines, msg)
	if len(c.logLines) > 12 {
		c.logLines = c.logLines[len(c.logLines)-12:]
	}
	c.logLabel.SetText(strings.Join(c.logLines, "\n"))
}

func (c *CardGameUI) queue(fn func()) {
	select {
	case c.eventQueue <- fn:
	default:
		go func() { c.eventQueue <- fn }()
	}
}

// layoutHand arranges player hand cards in a row with slight spacing.
func (c *CardGameUI) layoutHand() {
	if c.hand != nil {
		c.hand.Layout()
		return
	}
	if c.handArea == Control.Nil {
		return
	}
	x := float64(0)
	for _, view := range c.cardViews {
		if view.owner != c.player || view.location != "hand" || c.dragging == view {
			continue
		}
		view.homePos = Vector2.New(x, 0)
		view.button.AsControl().SetPosition(view.homePos)
		x += 140
	}
}

func (c *CardGameUI) viewportPosition(global Vector2.XY) Vector2.XY {
	if c.handContainer == SubViewportContainer.Nil || c.handViewport == SubViewport.Nil {
		return global
	}
	containerPos := c.handContainer.AsControl().GlobalPosition()
	containerSize := c.handContainer.AsControl().Size()
	vpSize := c.handViewport.Size()
	if containerSize.X == 0 || containerSize.Y == 0 {
		return global
	}
	x := (global.X - containerPos.X) * Float.X(vpSize.X) / containerSize.X
	y := (global.Y - containerPos.Y) * Float.X(vpSize.Y) / containerSize.Y
	return Vector2.XY{x, y}
}

func (c *CardGameUI) mapPointerTo3D(global Vector2.XY) Vector3.XYZ {
	if c.hand3d == nil || !c.hand3d.ready() {
		return Vector3.XYZ{}
	}
	local := c.viewportPosition(global)
	origin := c.hand3d.camera.ProjectRayOrigin(local)
	dir := c.hand3d.camera.ProjectRayNormal(local)
	if math.Abs(float64(dir.Y)) < 0.0001 {
		return origin
	}
	t := (Float.X(0.08) - origin.Y) / dir.Y
	if t < 0 {
		t = 0
	}
	return Vector3.XYZ{
		origin.X + dir.X*t,
		origin.Y + dir.Y*t,
		origin.Z + dir.Z*t,
	}
}

func (c *CardGameUI) dropCard3D(view *cardView, global Vector2.XY) {
	if c.board3d == nil || c.hand == nil || view == nil {
		return
	}
	world := c.mapPointerTo3D(global)
	if c.board3d.Place(view, world) {
		c.hand.Detach(view)
		return
	}
	// Snap back to hand
	view.location = "hand"
	view.fieldIndex = -1
}

func loadDeck() ([]*engine.Card, error) {
	parser := engine.NewCardParser()
	data, err := loadCardBytes()
	if err != nil {
		return nil, err
	}
	blocks := strings.Split(strings.TrimSpace(string(data)), "\n\n")
	deck := []*engine.Card{}
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		card, err := parser.Parse(block, true)
		if err != nil {
			continue
		}
		deck = append(deck, card)
	}
	if len(deck) == 0 {
		return nil, fmt.Errorf("no cards parsed from assets/cards.txt")
	}
	return deck, nil
}

// loadCardBytes tries common locations so the game works when run from the graphics project dir.
func loadCardBytes() ([]byte, error) {
	paths := []string{
		"assets/cards.txt",       // running from repo root
		"../assets/cards.txt",    // running from graphics/ subproject
		"../../assets/cards.txt", // running from graphics/.godot or similar
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("unable to locate assets/cards.txt (tried %v)", paths)
}

func describeChoice(choice any) string {
	switch v := choice.(type) {
	case *engine.CardInstance:
		return fmt.Sprintf("%s (%s/%s)", v.GetName(), v.GetPower().String(), v.GetHealth().String())
	case engine.Ability:
		return v.Text()
	case *engine.Player:
		return "Player"
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%T", v)
	}
}

func extractLife(text string) int {
	parts := strings.Split(text, ":")
	if len(parts) != 2 {
		return startingLife
	}
	val := strings.TrimSpace(parts[1])
	num, err := strconv.Atoi(val)
	if err != nil {
		return startingLife
	}
	return num
}

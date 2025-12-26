package godot

import (
	"github.com/SvenDH/go-card-engine/engine"

	"graphics.gd/classdb"
	"graphics.gd/classdb/BoxContainer"
	"graphics.gd/classdb/Button"
	"graphics.gd/classdb/Control"
	"graphics.gd/classdb/HBoxContainer"
	"graphics.gd/classdb/Input"
	"graphics.gd/classdb/InputEvent"
	"graphics.gd/classdb/InputEventMouseButton"
	"graphics.gd/classdb/InputEventMouseMotion"
	"graphics.gd/classdb/Label"
	"graphics.gd/classdb/SubViewport"
	"graphics.gd/classdb/SubViewportContainer"
	"graphics.gd/classdb/VBoxContainer"
	"graphics.gd/variant/Float"
	"graphics.gd/variant/Object"
	"graphics.gd/variant/Vector2"
	"graphics.gd/variant/Vector2i"
)

const (
	defaultBoardSlots = 5
	startingLife      = 20
)

// CardGameUI is a Godot/graphics.gd driven UI for the card engine.
// It listens for engine events and reflects them with simple controls and prompts.
type CardGameUI struct {
	Control.Extension[CardGameUI]

	handContainer SubViewportContainer.Instance
	handViewport  SubViewport.Instance
	phaseLabel    Label.Instance

	currentPrompt  engine.EventType
	promptPlayer   *engine.Player
	promptChoices  []any
	promptExpected int

	hand3d  *hand3DScene
	board3d *board3DScene

	eventQueue chan func()

	game   *engine.GameState
	player *engine.Player
	enemy  *engine.Player

	cardViews map[int]*cardView
	hand      *handScene
	turn      int
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

	skip := Button.New()
	skip.SetText("Skip")
	skip.AsBaseButton().OnPressed(func() {
		c.sendSelection(engine.SkipCode)
	})
	header.AsNode().AddChild(skip.AsNode())

	c.phaseLabel = Label.New()
	c.phaseLabel.SetText("Phase: --")
	header.AsNode().AddChild(c.phaseLabel.AsNode())

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
			c.updateHandRaise(gpos)
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
					c.selectCardView(view)
				}
				return
			}
			c.updateHandRaise(gpos)
		}
	})

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
			c.updateHandRaise(c.AsControl().AsCanvasItem().GetGlobalMousePosition())
			if c.hand3d != nil {
				c.hand3d.Update()
			}
			return
		}
	}
	c.updateHandRaise(c.AsControl().AsCanvasItem().GetGlobalMousePosition())
	if c.hand3d != nil {
		c.hand3d.Update()
	}
}

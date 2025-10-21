package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type MouseAction int

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMotion
	MouseEnter
	MouseLeave
	MouseDragMotion
	MouseDragRelease
	MouseDragEnter
	MouseDragLeave
)

type Msg interface{}

type Tick struct {
	DeltaTime float32 // Delta time in seconds
}

type MouseEvent struct {
	X, Y, RelX, RelY int
	Action           MouseAction
	Button           ebiten.MouseButton
	Zone             *Zone
}

type KeyEvent struct {
	Key     ebiten.Key
	Pressed bool // true for key press, false for key release
}

type Cmd func() Msg

func Batch(cmds ...Cmd) Cmd {
	return compactCmds[BatchMsg](cmds)
}

// BatchMsg is a message used to perform a bunch of commands concurrently with
// no ordering guarantees. You can send a BatchMsg with Batch.
type BatchMsg []Cmd

// Sequence runs the given commands one at a time, in order. Contrast this with
// Batch, which runs commands concurrently.
func Sequence(cmds ...Cmd) Cmd {
	return compactCmds[sequenceMsg](cmds)
}

// sequenceMsg is used internally to run the given commands in order.
type sequenceMsg []Cmd

// compactCmds ignores any nil commands in cmds, and returns the most direct
// command possible. That is, considering the non-nil commands, if there are
// none it returns nil, if there is exactly one it returns that command
// directly, else it returns the non-nil commands as type T.
func compactCmds[T ~[]Cmd](cmds []Cmd) Cmd {
	var validCmds []Cmd //nolint:prealloc
	for _, c := range cmds {
		if c == nil {
			continue
		}
		validCmds = append(validCmds, c)
	}
	switch len(validCmds) {
	case 0:
		return nil
	case 1:
		return validCmds[0]
	default:
		return func() Msg {
			return T(validCmds)
		}
	}
}

// Based on bubbletea model
type Model interface {
	Init() Cmd
	Update(msg Msg) (Model, Cmd)
	View() *Image
}

type Program struct {
	ctx                    context.Context
	M                      Model
	Width, Height          int
	ShowDebug              bool
	LastMouseX, LastMouseY int
	zones                  []*Zone
	initialized            bool
	dragCapture            *Zone
	draggedZone            *Zone
	lastDragHovered        *Zone
	lastUpdateTime         float64
}

// GetDraggedData returns the data being dragged, or nil if nothing is being dragged
func (p *Program) GetDraggedData() interface{} {
	if p.draggedZone != nil {
		return p.draggedZone.DragData
	}
	return nil
}

func (p *Program) Update() error {
	if !p.initialized {
		p.initialized = true
		p.runUpdate(p.M, p.M.Init())
	}
	mx, my := ebiten.CursorPosition()
	tmx, tmy := mx/TileSize, my/TileSize
	// First pass (reverse order): find topmost capturing zone
	topZoneIndex := -1
	for i := len(p.zones) - 1; i >= 0; i-- {
		p.zones[i].MouseX = tmx - p.zones[i].X
		p.zones[i].MouseY = tmy - p.zones[i].Y
		if p.zones[i].InBounds(tmx, tmy) && p.zones[i].Capture && topZoneIndex == -1 {
			topZoneIndex = i
		}
	}

	// Second pass: update all zones, but only allow hover for topmost
	for i := range p.zones {
		inBounds := p.zones[i].InBounds(tmx, tmy)
		hovered := p.zones[i].hovered

		// Only allow hover if this is the topmost capturing zone, or no capturing zone claimed it
		shouldHover := inBounds && (topZoneIndex == -1 || i == topZoneIndex)

		if shouldHover {
			p.zones[i].hovered = true
			if !hovered && p.zones[i].Owner != nil {
				// Dispatch directly to the zone owner
				p.runUpdate(p.zones[i].Owner, MouseEvent{
					X:      mx,
					Y:      my,
					RelX:   p.zones[i].MouseX,
					RelY:   p.zones[i].MouseY,
					Action: MouseEnter,
					Zone:   p.zones[i],
				})
			}
		} else {
			if hovered && p.zones[i].Owner != nil {
				// Dispatch directly to the zone owner
				p.runUpdate(p.zones[i].Owner, MouseEvent{
					X:      mx,
					Y:      my,
					RelX:   p.zones[i].MouseX,
					RelY:   p.zones[i].MouseY,
					Action: MouseLeave,
					Zone:   p.zones[i],
				})
			}
			p.zones[i].hovered = false
		}
		if p.zones[i].hovered {
			if mx != p.LastMouseX || my != p.LastMouseY {
				p.runUpdate(p.zones[i], MouseEvent{
					X:      mx,
					Y:      my,
					RelX:   tmx,
					RelY:   tmy,
					Action: MouseMotion,
				})
			}
			for j := range ebiten.MouseButtonMax {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(j)) {
					// Start drag capture for capturing zones
					if p.zones[i].Capture {
						p.dragCapture = p.zones[i]
					}
					// Track draggable zones
					if p.zones[i].Draggable {
						p.draggedZone = p.zones[i]
					}
					p.runUpdate(p.zones[i], MouseEvent{
						X:      mx,
						Y:      my,
						RelX:   tmx,
						RelY:   tmy,
						Action: MousePress,
						Button: ebiten.MouseButton(j),
					})
				}
				if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton(j)) {
					p.runUpdate(p.zones[i], MouseEvent{
						X:      mx,
						Y:      my,
						RelX:   tmx,
						RelY:   tmy,
						Action: MouseRelease,
						Button: ebiten.MouseButton(j),
					})
				}
			}
		}
	}

	// Handle drag capture - send drag-specific events to captured zone
	if p.dragCapture != nil {
		// Send drag motion events to captured zone
		if mx != p.LastMouseX || my != p.LastMouseY {
			p.runUpdate(p.dragCapture, MouseEvent{
				X:      mx,
				Y:      my,
				RelX:   tmx,
				RelY:   tmy,
				Action: MouseDragMotion,
			})
		}

		// Check for button release to end capture
		for j := range ebiten.MouseButtonMax {
			if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton(j)) {
				// Send release to the dragged zone
				p.runUpdate(p.dragCapture, MouseEvent{
					X:      mx,
					Y:      my,
					RelX:   tmx,
					RelY:   tmy,
					Action: MouseDragRelease,
					Button: ebiten.MouseButton(j),
				})

				// Send drop event to droppable zone under cursor (if any)
				if p.lastDragHovered != nil {
					p.runUpdate(p.lastDragHovered, MouseEvent{
						X:      mx,
						Y:      my,
						RelX:   tmx,
						RelY:   tmy,
						Action: MouseDragRelease,
						Button: ebiten.MouseButton(j),
						Zone:   p.draggedZone, // Pass dragged zone so drop target can access DragData
					})
				}

				// Release capture
				p.dragCapture = nil
				p.draggedZone = nil
				// Clear drag hover state
				if p.lastDragHovered != nil {
					p.lastDragHovered.dragHovered = false
					p.lastDragHovered = nil
				}
			}
		}

		// Update drag hover state for droppable zones
		if p.draggedZone != nil {
			var currentDragHovered *Zone
			// Find droppable zone under cursor (reverse order for top-most)
			for i := len(p.zones) - 1; i >= 0; i-- {
				if p.zones[i].Droppable && p.zones[i] != p.draggedZone && p.zones[i].InBounds(tmx, tmy) {
					currentDragHovered = p.zones[i]
					break
				}
			}

			// Handle drag enter/leave for droppable zones
			if currentDragHovered != p.lastDragHovered {
				if p.lastDragHovered != nil {
					p.lastDragHovered.dragHovered = false
					if p.lastDragHovered.Owner != nil {
						p.runUpdate(p.lastDragHovered.Owner, MouseEvent{
							X:      mx,
							Y:      my,
							RelX:   tmx,
							RelY:   tmy,
							Action: MouseDragLeave,
							Zone:   p.lastDragHovered,
						})
					}
				}
				if currentDragHovered != nil {
					currentDragHovered.dragHovered = true
					if currentDragHovered.Owner != nil {
						p.runUpdate(currentDragHovered.Owner, MouseEvent{
							X:      mx,
							Y:      my,
							RelX:   tmx,
							RelY:   tmy,
							Action: MouseDragEnter,
							Zone:   currentDragHovered,
						})
					}
				}
				p.lastDragHovered = currentDragHovered
			}
		}
	}
	p.LastMouseX = mx
	p.LastMouseY = my
	for i := range ebiten.KeyMax {
		if inpututil.IsKeyJustPressed(ebiten.Key(i)) {
			p.runUpdate(p.M, KeyEvent{Key: ebiten.Key(i), Pressed: true})
		}
		if inpututil.IsKeyJustReleased(ebiten.Key(i)) {
			p.runUpdate(p.M, KeyEvent{Key: ebiten.Key(i), Pressed: false})
		}
	}

	// Calculate delta time
	currentTime := float64(ebiten.ActualTPS())
	if currentTime == 0 {
		currentTime = 60 // Default to 60 TPS if not available
	}
	deltaTime := float32(1.0 / currentTime)

	p.runUpdate(p.M, Tick{DeltaTime: deltaTime})
	return nil
}

func (p *Program) Draw(screen *ebiten.Image) {
	tm := p.M.View()
	tm.Draw(screen, 0, 0)
	p.zones = tm.Zones
	if p.ShowDebug {
		msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.ActualTPS(), ebiten.ActualFPS())
		ebitenutil.DebugPrint(screen, msg)
	}
}

func (p *Program) Layout(outsideW, outsideH int) (int, int) {
	if p.Width == 0 && p.Height == 0 {
		tm := p.M.View()
		return tm.W * TileSize, tm.H * TileSize
	}
	return p.Width, p.Height
}

func (p *Program) runUpdate(m Model, msg Msg) {
	var cmd Cmd
	for {
		switch m := msg.(type) {
		case BatchMsg:
			go p.execBatchMsg(m)
			continue

		case sequenceMsg:
			go p.execSequenceMsg(m)
			continue
		}
		m, cmd = m.Update(msg)
		if cmd == nil {
			return
		}
		msg = cmd()
		if msg == nil {
			return
		}
	}
}

func (p *Program) execSequenceMsg(msg sequenceMsg) {
	// Execute commands one at a time, in order.
	for _, cmd := range msg {
		if cmd == nil {
			continue
		}
		msg := cmd()
		switch msg := msg.(type) {
		case BatchMsg:
			p.execBatchMsg(msg)
		case sequenceMsg:
			p.execSequenceMsg(msg)
		default:
			p.runUpdate(p.M, msg)
		}
	}
}

func (p *Program) execBatchMsg(msg BatchMsg) {
	// Execute commands one at a time.
	var wg sync.WaitGroup
	for _, cmd := range msg {
		if cmd == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			msg := cmd()
			switch msg := msg.(type) {
			case BatchMsg:
				p.execBatchMsg(msg)
			case sequenceMsg:
				p.execSequenceMsg(msg)
			default:
				p.runUpdate(p.M, msg)
			}
		}()
	}

	wg.Wait() // wait for all commands from batch msg to finish
}

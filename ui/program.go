package ui

import (
	"context"
	"fmt"

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
)

type Msg interface{}

type Tick struct{}

type MouseEvent struct {
	X, Y, RelX, RelY int
	Action           MouseAction
	Button           ebiten.MouseButton
	Zone             *Zone
}

type KeyEvent struct {
	Key ebiten.Key
}

type Cmd func() Msg

// Based on bubbletea model
type Model interface {
	Init() Cmd
	Update(msg Msg) (Model, Cmd)
	View() *TileMap
}

type Program struct {
	ctx                    context.Context
	M                      Model
	Width, Height          int
	ShowDebug              bool
	LastMouseX, LastMouseY int
	zones                  []*Zone
	initialized            bool
}

func (p *Program) Update() error {
	if !p.initialized {
		p.initialized = true
		p.runUpdate(p.M.Init())
	}
	mx, my := ebiten.CursorPosition()
	tmx, tmy := mx/TileSize, my/TileSize
	if mx != p.LastMouseX || my != p.LastMouseY {
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
					p.zones[i].Owner.Update(MouseEvent{
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
					p.zones[i].Owner.Update(MouseEvent{
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
		}
		p.runUpdate(MouseEvent{
			X:      mx,
			Y:      my,
			RelX:   tmx,
			RelY:   tmy,
			Action: MouseMotion,
		})
		p.LastMouseX = mx
		p.LastMouseY = my
	}
	for i := range ebiten.MouseButtonMax {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(i)) {
			p.runUpdate(MouseEvent{
				X:      mx,
				Y:      my,
				RelX:   tmx,
				RelY:   tmy,
				Action: MousePress,
				Button: ebiten.MouseButton(i),
			})
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton(i)) {
			p.runUpdate(MouseEvent{
				X:      mx,
				Y:      my,
				RelX:   tmx,
				RelY:   tmy,
				Action: MouseRelease,
				Button: ebiten.MouseButton(i),
			})
		}
	}
	for i := range ebiten.KeyMax {
		if inpututil.IsKeyJustPressed(ebiten.Key(i)) {
			p.runUpdate(KeyEvent{Key: ebiten.Key(i)})
		}
		if inpututil.IsKeyJustReleased(ebiten.Key(i)) {
			p.runUpdate(KeyEvent{Key: ebiten.Key(i)})
		}
	}
	p.runUpdate(Tick{})
	return nil
}

func (p *Program) runUpdate(msg Msg) {
	var cmd Cmd
	for {
		p.M, cmd = p.M.Update(msg)
		if cmd == nil {
			return
		}
		msg = cmd()
		if msg == nil {
			return
		}
	}
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

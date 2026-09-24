package players

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugins/world"
)

// Player is whoever acts in the game and may look at a part of it: a camera and the View through
// it, and — at this keyboard — the bindings that turn its input into commands.
type Player struct {
	ID     control.PlayerID
	Name   string
	Camera camera.Camera
	View   *world.View

	local    bool
	bindings []control.Binding
	cursor   geom.Vec
	held     map[ebiten.MouseButton]geom.Vec // buttons down and where they went down
}

// Bind adds bindings to the player; two on one Trigger are an error, never a silent last-one-wins.
func (p *Player) Bind(bindings ...control.Binding) error {
	for _, b := range bindings {
		if b.Command() == nil {
			return fmt.Errorf("players: %q is not a Binding built with control.Command", b.Label)
		}
		for _, have := range p.bindings {
			if have.Trigger == b.Trigger {
				return fmt.Errorf("players: %q and %q are both bound to %v for %s", have.Label, b.Label, b.Trigger, p.Name)
			}
		}
		p.bindings = append(p.bindings, b)
	}
	return nil
}

// Bindings lists what the player can do, in the order bound.
func (p *Player) Bindings() []control.Binding { return p.bindings }

// DragBox is the drag in progress of a button the player has a Drag binding on, in screen pixels.
func (p *Player) DragBox() (start, current geom.Vec, dragging bool) {
	for _, b := range p.bindings {
		d, ok := b.Trigger.(control.Drag)
		if !ok {
			continue
		}
		if at, down := p.held[d.Button]; down {
			return at, p.cursor, true
		}
	}
	return geom.Vec{}, geom.Vec{}, false
}

// screen is the window's size in pixels, as the camera shows the world.
func (p *Player) screen() geom.Vec {
	b := p.Camera.Bounds()
	z := float64(p.Camera.Zoom())
	return geom.NewVec((b.BottomRight.X-b.TopLeft.X)*z, (b.BottomRight.Y-b.TopLeft.Y)*z)
}

func (p *Player) press(button ebiten.MouseButton, at geom.Vec) {
	if p.held == nil {
		p.held = map[ebiten.MouseButton]geom.Vec{}
	}
	p.held[button] = at
}

// release forgets the button and reports where it went down, if it was down.
func (p *Player) release(button ebiten.MouseButton) (geom.Vec, bool) {
	at, down := p.held[button]
	delete(p.held, button)
	return at, down
}

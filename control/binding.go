package control

import (
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/camera"
)

// Mods is the modifier keys a trigger asks for; a trigger fires only with exactly these held.
type Mods struct{ Shift, Ctrl, Alt bool }

// Trigger is what fires a Binding: a key, a button, a gesture. The concrete triggers are values,
// so two bindings on one trigger are told apart when bound.
type Trigger interface{ trigger() }

// KeyPress fires when Key goes down with Mods held.
type KeyPress struct {
	Key  ebiten.Key
	Mods Mods
}

// ButtonPress fires when Button goes down with Mods held; Context.Cursor is where.
type ButtonPress struct {
	Button ebiten.MouseButton
	Mods   Mods
}

// Drag fires when Button comes up with Mods held, after going down: Context.Start is where it went
// down, Context.Cursor where it came up. A click is a Drag of no length.
type Drag struct {
	Button ebiten.MouseButton
	Mods   Mods
}

// Wheel fires on any scroll; Context.Wheel is how much.
type Wheel struct{}

// ButtonHeld fires every tick Button is down and the cursor moved inside the window;
// Context.Delta is by how much.
type ButtonHeld struct{ Button ebiten.MouseButton }

// CursorAtEdge fires every tick the cursor rests near a window edge; the carrier says how near.
type CursorAtEdge struct{}

func (KeyPress) trigger()     {}
func (ButtonPress) trigger()  {}
func (Drag) trigger()         {}
func (Wheel) trigger()        {}
func (ButtonHeld) trigger()   {}
func (CursorAtEdge) trigger() {}

// Context is what a binding builds its command from: the player, its camera and this tick's input
// in screen pixels; World and WorldBox go through the camera.
type Context struct {
	Player      PlayerID
	Camera      camera.Camera
	Cursor      geom.Vec // where the cursor is, or where a button went down or up
	Start       geom.Vec // where a Drag began
	Delta       geom.Vec // cursor movement this tick
	Wheel       float64
	Screen      geom.Vec // the window's size in pixels
	Mods        Mods
	FillsScreen bool
}

// World is the world point under screen position s.
func (c Context) World(s geom.Vec) geom.Vec {
	x, y := c.Camera.FromScreen(float32(s.X), float32(s.Y))
	return geom.NewVec(float64(x), float64(y))
}

// WorldBox is the world rectangle between screen points a and b, at least one unit a side and no
// wider than what was dragged even across a wrapping seam.
func (c Context) WorldBox(a, b geom.Vec) geom.AABB {
	x0, y0, x1, y1 := camera.FromScreenRect(c.Camera, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y))
	minX, maxX := float64(min(x0, x1)), float64(max(x0, x1))
	minY, maxY := float64(min(y0, y1)), float64(max(y0, y1))
	return geom.NewAABBAt(geom.NewVec(minX, minY), max(maxX-minX, 1), max(maxY-minY, 1))
}

// Binding is one thing a player can do: a Trigger, the command it issues and a label saying what
// it does, for a help screen. Build one with Command.
type Binding struct {
	Trigger Trigger
	Label   string

	command reflect.Type
	build   func(Context) (any, bool)
}

// Command is a Binding issuing a C built from the Context when trigger fires; build may decline.
func Command[C any](trigger Trigger, label string, build func(c Context) (C, bool)) Binding {
	return Binding{Trigger: trigger, Label: label, command: reflect.TypeFor[C](), build: func(c Context) (any, bool) {
		cmd, ok := build(c)
		return cmd, ok
	}}
}

// Command is the type of command the binding issues; nil for one not built with Command.
func (b Binding) Command() reflect.Type { return b.command }

// Build is the command for c, if the binding has one to give.
func (b Binding) Build(c Context) (any, bool) {
	if b.build == nil {
		return nil, false
	}
	return b.build(c)
}

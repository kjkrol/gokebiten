package control

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokg/geom"
)

type KeyAction int

const (
	ActionPress KeyAction = iota
	ActionRelease
)

type KeyEvent struct {
	Key    ebiten.Key
	Action KeyAction
}

type ClickEvent struct {
	Pos    geom.Vec[int32]
	Button ebiten.MouseButton
	Action KeyAction
}

type InputEvents struct {
	MousePos    geom.Vec[int32]
	CursorDelta geom.Vec[int32]
	Modifiers   struct {
		Shift, Ctrl, Alt bool
	}

	ClickQueue  []ClickEvent
	KeyEvents   []KeyEvent
	ScrollDelta float64
}

func (*InputEvents) PluginResource() {}

func (e *InputEvents) ResetTransient() {
	e.ClickQueue = e.ClickQueue[:0]
	e.KeyEvents = e.KeyEvents[:0]
	e.ScrollDelta = 0
	e.CursorDelta = geom.Vec[int32]{}
}

func (e *InputEvents) AddKeyEvent(key ebiten.Key, action KeyAction) {
	e.KeyEvents = append(e.KeyEvents, KeyEvent{Key: key, Action: action})
}

// AddClickEvent takes plain int for caller ergonomics — stored as geom.Vec[int32].
func (e *InputEvents) AddClickEvent(x, y int, button ebiten.MouseButton, action KeyAction) {
	e.ClickQueue = append(e.ClickQueue, ClickEvent{
		Pos:    geom.NewVec(int32(x), int32(y)),
		Button: button,
		Action: action,
	})
}

type InputAdapter interface {
	Capture(e *InputEvents)
}

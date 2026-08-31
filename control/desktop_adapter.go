package control

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/kjkrol/gokg/geom"
)

type DesktopAdapter struct{}

func (a *DesktopAdapter) Capture(e *InputEvents) {
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		e.AddKeyEvent(k, ActionPress)
	}
	for _, k := range inpututil.AppendJustReleasedKeys(nil) {
		e.AddKeyEvent(k, ActionRelease)
	}

	currX, currY := ebiten.CursorPosition()
	next := geom.NewVec(int32(currX), int32(currY))
	e.CursorDelta = geom.NewVec(next.X-e.MousePos.X, next.Y-e.MousePos.Y)
	e.MousePos = next

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		e.AddClickEvent(currX, currY, ebiten.MouseButtonLeft, ActionPress)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		e.AddClickEvent(currX, currY, ebiten.MouseButtonLeft, ActionRelease)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		e.AddClickEvent(currX, currY, ebiten.MouseButtonRight, ActionPress)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		e.AddClickEvent(currX, currY, ebiten.MouseButtonRight, ActionRelease)
	}

	e.Modifiers.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	e.Modifiers.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl)
}

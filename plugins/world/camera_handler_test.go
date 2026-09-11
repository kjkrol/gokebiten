package world

import (
	"testing"

	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokg/geom"
)

func TestDefaultCameraHandler_Scroll_Zooms(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)

	h.HandleEvents(&control.InputEvents{ScrollDelta: 1})
	if cam.Zoom() <= 1 {
		t.Fatalf("Zoom() after positive scroll = %v, want > 1", cam.Zoom())
	}

	zoomedIn := cam.Zoom()
	h.HandleEvents(&control.InputEvents{ScrollDelta: -1})
	if cam.Zoom() >= zoomedIn {
		t.Fatalf("Zoom() after negative scroll = %v, want < %v", cam.Zoom(), zoomedIn)
	}
}

func TestDefaultCameraHandler_MiddleDrag_Pans(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	cam.ZoomIn(2, 500, 500) // zoom in first: a viewport equal to the world has no room to pan otherwise
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MiddleDown: true, CursorDelta: geom.NewVec[int32](10, 0)})

	after := cam.Bounds()
	if after.TopLeft.X != before.TopLeft.X-10 {
		t.Errorf("Bounds().TopLeft.X after middle-drag = %d, want %d", after.TopLeft.X, before.TopLeft.X-10)
	}
}

func TestDefaultCameraHandler_CursorNearRightEdge_ScrollsRight(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	cam.ZoomIn(2, 500, 500) // zoom in first: a viewport equal to the world has no room to pan otherwise
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec[int32](990, 500)})

	after := cam.Bounds()
	if after.TopLeft.X <= before.TopLeft.X {
		t.Errorf("Bounds().TopLeft.X after cursor near right edge = %d, want > %d", after.TopLeft.X, before.TopLeft.X)
	}
}

// TestDefaultCameraHandler_CursorAtTrueEdge_DeadZoneDoesNotScroll guards
// the actual reported bug: on platforms where the cursor's reported
// position freezes at the true screen edge once it leaves the window
// (e.g. Wayland, where no further motion events arrive), the dead zone
// carved out of the hot-scroll band must exclude that frozen position,
// or edge-scroll continues forever even after the cursor has left.
func TestDefaultCameraHandler_CursorAtTrueEdge_DeadZoneDoesNotScroll(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	cam.ZoomIn(2, 500, 500) // zoom in first: a viewport equal to the world has no room to pan otherwise
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec[int32](999, 500)})

	after := cam.Bounds()
	if after != before {
		t.Errorf("Bounds() after cursor frozen at the true right edge = %+v, want unchanged %+v", after, before)
	}
}

// TestDefaultCameraHandler_CursorAtTrueEdge_WindowFillsScreen_Scrolls
// guards that the dead zone only applies to a bordered, smaller-than-
// screen window: once the window fills the whole screen (fullscreen, or
// a borderless window matching the monitor), there's no desktop beyond
// the edge for the cursor to freeze against, so the hot zone should
// reach all the way to the true edge again.
func TestDefaultCameraHandler_CursorAtTrueEdge_WindowFillsScreen_Scrolls(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	cam.ZoomIn(2, 500, 500) // zoom in first: a viewport equal to the world has no room to pan otherwise
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec[int32](999, 500), WindowFillsScreen: true})

	after := cam.Bounds()
	if after.TopLeft.X <= before.TopLeft.X {
		t.Errorf("Bounds().TopLeft.X after cursor at the true edge with WindowFillsScreen = %d, want > %d", after.TopLeft.X, before.TopLeft.X)
	}
}

// TestDefaultCameraHandler_MiddleDrag_CursorOutsideWindow_DoesNotPan
// guards that leaving the window mid-drag stops the pan too, not just
// edge-scroll — MiddleDown was handled (and returned) before the
// in-window check used to run, so a drag that carried the cursor outside
// the window kept panning off whatever stale/out-of-range CursorDelta
// ebiten reported.
func TestDefaultCameraHandler_MiddleDrag_CursorOutsideWindow_DoesNotPan(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	cam.ZoomIn(2, 500, 500) // zoom in first: a viewport equal to the world has no room to pan otherwise
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{
		MiddleDown:  true,
		MousePos:    geom.NewVec[int32](-5, 500),
		CursorDelta: geom.NewVec[int32](10, 0),
	})

	after := cam.Bounds()
	if after != before {
		t.Errorf("Bounds() after middle-drag with cursor outside the window = %+v, want unchanged %+v", after, before)
	}
}

// TestDefaultCameraHandler_CursorOutsideWindow_DoesNotScroll guards the
// reported bug: ebiten.CursorPosition() (surfaced via events.MousePos)
// reports coordinates outside [0,screenW)x[0,screenH) once the cursor
// leaves the game window — the old margin check treated "far outside the
// window" the same as "near an in-window edge" and kept scrolling forever.
func TestDefaultCameraHandler_CursorOutsideWindow_DoesNotScroll(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, false)
	cam.ZoomIn(2, 500, 500) // zoom in first: a viewport equal to the world has no room to pan otherwise
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec[int32](-5, 500)})

	after := cam.Bounds()
	if after != before {
		t.Errorf("Bounds() after cursor outside the window = %+v, want unchanged %+v", after, before)
	}
}

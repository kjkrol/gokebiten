package world

import (
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
)

func TestDefaultCameraHandler_Scroll_Zooms(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, 0)
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
	cam := camera.NewFromSpace(1000, 1000, 0)
	cam.ZoomIn(2, 500, 500)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MiddleDown: true, CursorDelta: geom.NewVec(10, 0)})

	after := cam.Bounds()
	if after.TopLeft.X != before.TopLeft.X-10 {
		t.Errorf("Bounds().TopLeft.X after middle-drag = %v, want %v", after.TopLeft.X, before.TopLeft.X-10)
	}
}

func TestDefaultCameraHandler_CursorNearRightEdge_ScrollsRight(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, 0)
	cam.ZoomIn(2, 500, 500)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec(990, 500)})

	after := cam.Bounds()
	if after.TopLeft.X <= before.TopLeft.X {
		t.Errorf("Bounds().TopLeft.X after cursor near right edge = %v, want > %v", after.TopLeft.X, before.TopLeft.X)
	}
}

func TestDefaultCameraHandler_CursorAtTrueEdge_DeadZoneDoesNotScroll(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, 0)
	cam.ZoomIn(2, 500, 500)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec(999, 500)})

	after := cam.Bounds()
	if after != before {
		t.Errorf("Bounds() after cursor frozen at the true right edge = %+v, want unchanged %+v", after, before)
	}
}

func TestDefaultCameraHandler_CursorAtTrueEdge_WindowFillsScreen_Scrolls(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, 0)
	cam.ZoomIn(2, 500, 500)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec(999, 500), WindowFillsScreen: true})

	after := cam.Bounds()
	if after.TopLeft.X <= before.TopLeft.X {
		t.Errorf("Bounds().TopLeft.X after cursor at the true edge with WindowFillsScreen = %v, want > %v", after.TopLeft.X, before.TopLeft.X)
	}
}

func TestDefaultCameraHandler_MiddleDrag_CursorOutsideWindow_DoesNotPan(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, 0)
	cam.ZoomIn(2, 500, 500)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{
		MiddleDown:  true,
		MousePos:    geom.NewVec(-5, 500),
		CursorDelta: geom.NewVec(10, 0),
	})

	after := cam.Bounds()
	if after != before {
		t.Errorf("Bounds() after middle-drag with cursor outside the window = %+v, want unchanged %+v", after, before)
	}
}

func TestDefaultCameraHandler_CursorOutsideWindow_DoesNotScroll(t *testing.T) {
	cam := camera.NewFromSpace(1000, 1000, 0)
	cam.ZoomIn(2, 500, 500)
	h := newDefaultCameraHandler(cam, defaultCameraScrollSpeed)
	before := cam.Bounds()

	h.HandleEvents(&control.InputEvents{MousePos: geom.NewVec(-5, 500)})

	after := cam.Bounds()
	if after != before {
		t.Errorf("Bounds() after cursor outside the window = %+v, want unchanged %+v", after, before)
	}
}

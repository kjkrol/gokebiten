package world

import (
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
)

const (
	defaultCameraEdgeMarginPixels   = 30
	defaultCameraEdgeDeadZonePixels = 10
	defaultCameraScrollSpeed        = 8
	defaultCameraZoomFactor         = 1.1
)

type defaultCameraHandler struct {
	cam         camera.Camera
	scrollSpeed int32
}

func newDefaultCameraHandler(cam camera.Camera, scrollSpeed int32) *defaultCameraHandler {
	return &defaultCameraHandler{cam: cam, scrollSpeed: scrollSpeed}
}

var _ control.EventHandler = (*defaultCameraHandler)(nil)

func (h *defaultCameraHandler) HandleEvents(events *control.InputEvents) {
	wx, wy := h.cam.FromScreen(float32(events.MousePos.X), float32(events.MousePos.Y))

	if events.ScrollDelta > 0 {
		h.cam.ZoomIn(defaultCameraZoomFactor, wx, wy)
	} else if events.ScrollDelta < 0 {
		h.cam.ZoomOut(defaultCameraZoomFactor, wx, wy)
	}

	bounds := h.cam.Bounds()
	screenW := float32(bounds.BottomRight.X-bounds.TopLeft.X) * h.cam.Zoom()
	screenH := float32(bounds.BottomRight.Y-bounds.TopLeft.Y) * h.cam.Zoom()

	mx, my := float32(events.MousePos.X), float32(events.MousePos.Y)
	if mx < 0 || mx >= screenW || my < 0 || my >= screenH {
		return
	}

	if events.MiddleDown {
		h.cam.Translate(-events.CursorDelta.X, -events.CursorDelta.Y)
		return
	}

	deadZone := float32(defaultCameraEdgeDeadZonePixels)
	if events.WindowFillsScreen {
		deadZone = 0
	}

	var dx, dy int32
	switch {
	case mx >= deadZone && mx < defaultCameraEdgeMarginPixels:
		dx = -h.scrollSpeed
	case mx <= screenW-deadZone && mx > screenW-defaultCameraEdgeMarginPixels:
		dx = h.scrollSpeed
	}
	switch {
	case my >= deadZone && my < defaultCameraEdgeMarginPixels:
		dy = -h.scrollSpeed
	case my <= screenH-deadZone && my > screenH-defaultCameraEdgeMarginPixels:
		dy = h.scrollSpeed
	}
	if dx != 0 || dy != 0 {
		h.cam.Translate(dx, dy)
	}
}

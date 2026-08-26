package render

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

func testViewport(x, y, w, h uint32) AABB {
	return geom.NewAABBAt(geom.NewVec(x, y), w, h)
}

// TestBasicCamera_Translate_WrapsOnToroidalWorld guards that a negative delta wraps at the world's size, not 2^32.
func TestBasicCamera_Translate_WrapsOnToroidalWorld(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](20, 20)
	c := NewBasicCamera(surface, testViewport(2, 2, 4, 4))

	c.Translate(-10, 0)

	if c.Viewport.TopLeft.X != 12 {
		t.Errorf("Viewport.TopLeft.X = %d, want 12 (wrapped: (2-10) mod 20)", c.Viewport.TopLeft.X)
	}
	if c.Viewport.TopLeft.Y != 2 {
		t.Errorf("Viewport.TopLeft.Y = %d, want 2 (unchanged)", c.Viewport.TopLeft.Y)
	}
}

// TestBasicCamera_Translate_ClampsOnEuclideanWorld guards that a non-toroidal world clamps instead of wrapping.
func TestBasicCamera_Translate_ClampsOnEuclideanWorld(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](20, 20)
	c := NewBasicCamera(surface, testViewport(2, 2, 4, 4))

	c.Translate(-10, 0)

	if c.Viewport.TopLeft.X != 0 {
		t.Errorf("Viewport.TopLeft.X = %d, want 0 (clamped, not wrapped)", c.Viewport.TopLeft.X)
	}
}

// TestBasicCamera_MoveTo_WrapsOnToroidalWorld guards that MoveTo's signed-delta computation is correct.
func TestBasicCamera_MoveTo_WrapsOnToroidalWorld(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](20, 20)
	c := NewBasicCamera(surface, testViewport(2, 2, 4, 4))

	c.MoveTo(18, 2)
	c.Translate(-10, 0)
	if c.Viewport.TopLeft.X != 8 {
		t.Fatalf("sanity check failed: Viewport.TopLeft.X = %d, want 8", c.Viewport.TopLeft.X)
	}

	c.Translate(-10, 0)
	if c.Viewport.TopLeft.X != 18 {
		t.Errorf("Viewport.TopLeft.X = %d, want 18 (wrapped: (8-10) mod 20)", c.Viewport.TopLeft.X)
	}
}

// TestBasicCamera_ExtendedBounds_MarginLargerThanPosition guards that a large margin doesn't underflow uint32.
func TestBasicCamera_ExtendedBounds_MarginLargerThanPosition(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](20, 20)
	c := NewBasicCamera(surface, testViewport(2, 2, 4, 4))

	got := c.ExtendedBounds(10)

	if got.TopLeft.X != 0 || got.TopLeft.Y != 0 {
		t.Errorf("ExtendedBounds(10).TopLeft = (%d,%d), want (0,0) (clamped, not underflowed)", got.TopLeft.X, got.TopLeft.Y)
	}
	if got.BottomRight.X > 20 || got.BottomRight.Y > 20 {
		t.Errorf("ExtendedBounds(10).BottomRight = (%d,%d), want within world bounds [0,20]", got.BottomRight.X, got.BottomRight.Y)
	}
}

func TestBasicCamera_Bounds_MatchesConstructedViewportAtZoom1(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](100, 100)
	viewport := testViewport(10, 10, 30, 20)
	c := NewBasicCamera(surface, viewport)

	got := c.Bounds()
	if got.TopLeft != viewport.TopLeft || got.BottomRight != viewport.BottomRight {
		t.Errorf("Bounds() = %+v, want %+v", got, viewport)
	}
}

func TestBasicCamera_ToScreen_NoZoomIsPlainOffset(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](100, 100)
	c := NewBasicCamera(surface, testViewport(10, 10, 30, 20))

	x, y := c.ToScreen(15, 12)
	if x != 5 || y != 2 {
		t.Errorf("ToScreen(15,12) = (%v,%v), want (5,2)", x, y)
	}
}

func TestBasicCamera_ZoomIn_ScalesToScreen(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := NewBasicCamera(surface, testViewport(100, 100, 100, 100))

	c.ZoomIn(2)
	if c.Zoom() != 2 {
		t.Fatalf("Zoom() = %v, want 2", c.Zoom())
	}

	x, y := c.ToScreen(125, 125)
	if x != 0 || y != 0 {
		t.Errorf("ToScreen(125,125) at zoom 2 = (%v,%v), want (0,0)", x, y)
	}

	x2, _ := c.ToScreen(126, 125)
	if x2 != 2 {
		t.Errorf("ToScreen(126,125) at zoom 2 = %v, want 2 (scale applied)", x2)
	}
}

func TestBasicCamera_ZoomIn_ThenZoomOut_RoundTrips(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := NewBasicCamera(surface, testViewport(100, 100, 100, 100))

	c.ZoomIn(2)
	c.ZoomOut(2)

	if c.Zoom() != 1 {
		t.Errorf("Zoom() after ZoomIn(2);ZoomOut(2) = %v, want 1", c.Zoom())
	}
}

func TestBasicCamera_ImplementsCameraInterface(t *testing.T) {
	var _ Camera = (*BasicCamera)(nil)
}

// TestBasicCamera_GobRoundTrip_SilentlyDropsUnexportedState documents that gob drops unexported fields without erroring.
func TestBasicCamera_GobRoundTrip_SilentlyDropsUnexportedState(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := NewBasicCamera(surface, testViewport(10, 10, 100, 100))
	c.ZoomIn(2)

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(c); err != nil {
		t.Fatalf("Encode: %v (expected this to silently succeed, not error)", err)
	}

	var decoded BasicCamera
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.Viewport != c.Viewport {
		t.Errorf("Viewport survived incorrectly: got %+v, want %+v", decoded.Viewport, c.Viewport)
	}
	if decoded.Zoom() != 0 {
		t.Errorf("Zoom() after gob round-trip = %v, want 0 (unexported field silently dropped, not preserved)", decoded.Zoom())
	}
	if got := decoded.Bounds(); got != (AABB{}) {
		t.Errorf("Bounds() after gob round-trip = %+v, want zero value (effective was never restored)", got)
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected Translate on a gob-decoded BasicCamera to panic (surface is nil), but it didn't")
			}
		}()
		decoded.Translate(1, 1)
	}()
}

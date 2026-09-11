package camera

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
	c := newBasicCamera(surface, testViewport(2, 2, 4, 4), true)

	c.Translate(-10, 0)

	if c.effective.TopLeft.X != 12 {
		t.Errorf("Viewport.TopLeft.X = %d, want 12 (wrapped: (2-10) mod 20)", c.effective.TopLeft.X)
	}
	if c.effective.TopLeft.Y != 2 {
		t.Errorf("Viewport.TopLeft.Y = %d, want 2 (unchanged)", c.effective.TopLeft.Y)
	}
}

// TestBasicCamera_Translate_ClampsOnEuclideanWorld guards that a non-toroidal world clamps instead of wrapping.
func TestBasicCamera_Translate_ClampsOnEuclideanWorld(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](20, 20)
	c := newBasicCamera(surface, testViewport(2, 2, 4, 4), false)

	c.Translate(-10, 0)

	if c.effective.TopLeft.X != 0 {
		t.Errorf("Viewport.TopLeft.X = %d, want 0 (clamped, not wrapped)", c.effective.TopLeft.X)
	}
}

// TestBasicCamera_MoveTo_WrapsOnToroidalWorld guards that MoveTo's signed-delta computation is correct.
func TestBasicCamera_MoveTo_WrapsOnToroidalWorld(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](20, 20)
	c := newBasicCamera(surface, testViewport(2, 2, 4, 4), true)

	c.MoveTo(18, 2)
	c.Translate(-10, 0)
	if c.effective.TopLeft.X != 8 {
		t.Fatalf("sanity check failed: Viewport.TopLeft.X = %d, want 8", c.effective.TopLeft.X)
	}

	c.Translate(-10, 0)
	if c.effective.TopLeft.X != 18 {
		t.Errorf("Viewport.TopLeft.X = %d, want 18 (wrapped: (8-10) mod 20)", c.effective.TopLeft.X)
	}
}

func TestBasicCamera_Bounds_MatchesConstructedViewportAtZoom1(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](100, 100)
	viewport := testViewport(10, 10, 30, 20)
	c := newBasicCamera(surface, viewport, false)

	got := c.Bounds()
	if got.TopLeft != viewport.TopLeft || got.BottomRight != viewport.BottomRight {
		t.Errorf("Bounds() = %+v, want %+v", got, viewport)
	}
}

func TestBasicCamera_ToScreen_NoZoomIsPlainOffset(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](100, 100)
	c := newBasicCamera(surface, testViewport(10, 10, 30, 20), false)

	x, y := c.ToScreen(15, 12)
	if x != 5 || y != 2 {
		t.Errorf("ToScreen(15,12) = (%v,%v), want (5,2)", x, y)
	}
}

func TestBasicCamera_ZoomIn_ScalesToScreen(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(100, 100, 100, 100), false)

	c.ZoomIn(2, 150, 150)
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
	c := newBasicCamera(surface, testViewport(100, 100, 100, 100), false)

	c.ZoomIn(2, 150, 150)
	c.ZoomOut(2, 150, 150)

	if c.Zoom() != 1 {
		t.Errorf("Zoom() after ZoomIn(2);ZoomOut(2) = %v, want 1", c.Zoom())
	}
}

// TestBasicCamera_Translate_ClampsEffectiveWhenZoomedIn guards that
// panning clamps against the zoomed-in (effective) size, not Viewport's
// own fixed reference size — a viewport already equal to the world
// still has room to pan once zoomed in.
func TestBasicCamera_Translate_ClampsEffectiveWhenZoomedIn(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 1000, 1000), false)
	c.ZoomIn(2, 500, 500) // effective = (250,250)-(750,750)

	c.Translate(1000, 0)

	b := c.Bounds()
	if b.BottomRight.X != 1000 {
		t.Errorf("Bounds().BottomRight.X = %d, want 1000 (clamped to world edge)", b.BottomRight.X)
	}
	if w := b.BottomRight.X - b.TopLeft.X; w != 500 {
		t.Errorf("effective width after clamped Translate = %d, want 500 (must not shrink)", w)
	}
}

// TestBasicCamera_ZoomOut_CappedToWorldFit guards that zooming out never
// reveals area beyond the world on either axis.
func TestBasicCamera_ZoomOut_CappedToWorldFit(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 1000, 1000), false)

	c.ZoomOut(10, 500, 500)

	if c.Zoom() < 1 {
		t.Errorf("Zoom() = %v, want >= 1 (viewport already equals world size)", c.Zoom())
	}
	if b := c.Bounds(); b.TopLeft.X != 0 || b.TopLeft.Y != 0 || b.BottomRight.X != 1000 || b.BottomRight.Y != 1000 {
		t.Errorf("Bounds() = %+v, want exactly (0,0)-(1000,1000) (capped at world fit)", b)
	}
}

// TestBasicCamera_ZoomIn_AnchorStaysUnderCursor guards that a
// non-centered anchor lands back at the same screen position after
// zooming — a centered anchor can't catch a sign error (delta is 0
// either way).
func TestBasicCamera_ZoomIn_AnchorStaysUnderCursor(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 1000, 1000), false)

	const anchorX, anchorY = 900, 500
	beforeX, beforeY := c.ToScreen(anchorX, anchorY)

	c.ZoomIn(1.1, anchorX, anchorY)

	afterX, afterY := c.ToScreen(anchorX, anchorY)
	const tol = 1.5 // int32 truncation in the correction step
	if diff := afterX - beforeX; diff < -tol || diff > tol {
		t.Errorf("ToScreen(anchor).X after ZoomIn = %v, want ~%v (anchor should stay under the cursor)", afterX, beforeX)
	}
	if diff := afterY - beforeY; diff < -tol || diff > tol {
		t.Errorf("ToScreen(anchor).Y after ZoomIn = %v, want ~%v (anchor should stay under the cursor)", afterY, beforeY)
	}
}

// TestBasicCamera_ZoomIn_CappedByMaxZoom guards that SetMaxZoom stops
// ZoomIn from magnifying further, even when the requested factor would
// otherwise exceed it.
func TestBasicCamera_ZoomIn_CappedByMaxZoom(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 100, 100), false)
	c.SetMaxZoom(2)

	c.ZoomIn(10, 50, 50)

	if c.Zoom() != 2 {
		t.Errorf("Zoom() = %v, want 2 (capped by SetMaxZoom)", c.Zoom())
	}
}

// TestBasicCamera_ZoomOut_CappedByConfiguredMinZoom guards that
// SetMinZoom raises the floor above the automatic world-fit one when
// the world is much larger than the viewport.
func TestBasicCamera_ZoomOut_CappedByConfiguredMinZoom(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](10000, 10000)
	c := newBasicCamera(surface, testViewport(0, 0, 100, 100), false)
	c.SetMinZoom(0.5)

	c.ZoomOut(100, 50, 50)

	if c.Zoom() != 0.5 {
		t.Errorf("Zoom() = %v, want 0.5 (capped by SetMinZoom, stricter than the world-fit floor)", c.Zoom())
	}
}

// TestNewFromSpaceWithConfig_AppliesViewportAndZoomLimits guards that
// the convenience constructor actually applies every Config field.
func TestNewFromSpaceWithConfig_AppliesViewportAndZoomLimits(t *testing.T) {
	c := NewFromSpaceWithConfig(1000, 1000, false, Config{
		ViewportWidth: 100, ViewportHeight: 100,
		MinZoom: 0.5,
		MaxZoom: 2,
	})

	if w := c.Bounds().BottomRight.X - c.Bounds().TopLeft.X; w != 100 {
		t.Errorf("Bounds() width = %d, want 100 (ViewportWidth applied)", w)
	}

	c.ZoomIn(10, 50, 50)
	if c.Zoom() != 2 {
		t.Errorf("Zoom() after ZoomIn(10,...) = %v, want 2 (MaxZoom applied)", c.Zoom())
	}

	c.ZoomOut(100, 50, 50)
	if c.Zoom() != 0.5 {
		t.Errorf("Zoom() after ZoomOut(100,...) = %v, want 0.5 (MinZoom applied)", c.Zoom())
	}
}

func TestBasicCamera_ImplementsCameraInterface(t *testing.T) {
	var _ Camera = (*basicCamera)(nil)
}

func TestBasicCamera_FromScreen_InvertsToScreen(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(100, 100, 200, 200), false)

	cases := []struct {
		name string
		zoom float32
		x, y float32
	}{
		{"no zoom, origin", 1, 100, 100},
		{"no zoom, interior point", 1, 150, 180},
		{"zoomed in", 2.5, 200, 220},
		{"zoomed out", 0.5, 120, 260},
	}

	for _, c2 := range cases {
		t.Run(c2.name, func(t *testing.T) {
			c.ZoomIn(c2.zoom/c.Zoom(), 200, 200)

			sx, sy := c.ToScreen(c2.x, c2.y)
			gotX, gotY := c.FromScreen(sx, sy)

			const tol = 0.01
			if diff := gotX - c2.x; diff < -tol || diff > tol {
				t.Errorf("FromScreen(ToScreen(%v,%v)).X = %v, want ~%v", c2.x, c2.y, gotX, c2.x)
			}
			if diff := gotY - c2.y; diff < -tol || diff > tol {
				t.Errorf("FromScreen(ToScreen(%v,%v)).Y = %v, want ~%v", c2.x, c2.y, gotY, c2.y)
			}
		})
	}
}

// TestBasicCamera_ToScreen_ToroidalWrapsWhenViewportFillsWorld guards the
// case a "nearest representative" wrap formula gets wrong: once the
// view's size equals the world's size (as in collision-demo), a point
// just behind the camera's reference must wrap to the far end of the
// view, not stay just-before-the-start.
func TestBasicCamera_ToScreen_ToroidalWrapsWhenViewportFillsWorld(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](1024, 1024)
	c := newBasicCamera(surface, testViewport(0, 0, 1024, 1024), true)
	c.Translate(8, 8) // effective.TopLeft is now (8,8)

	x, y := c.ToScreen(4, 4) // just behind the new reference point

	const wantX, wantY = 1020, 1020 // wraps to the far end: (4-8) mod 1024 = 1020
	if x != wantX || y != wantY {
		t.Errorf("ToScreen(4,4) = (%v,%v), want (%v,%v)", x, y, wantX, wantY)
	}
}

// TestBasicCamera_ToScreen_ToroidalWrapsNearCurrentView guards that an
// entity on the far side of a toroidal world's wrap seam is placed as a
// continuation of the current view, not jumped to the opposite side.
func TestBasicCamera_ToScreen_ToroidalWrapsNearCurrentView(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 200, 200), true)
	c.MoveTo(900, 900)

	nearX, nearY := c.ToScreen(950, 950)
	wrappedX, wrappedY := c.ToScreen(50, 50)

	if wrappedX <= nearX {
		t.Errorf("ToScreen(50,_).X = %v, want > ToScreen(950,_).X = %v (continues past the seam)", wrappedX, nearX)
	}
	if wrappedY <= nearY {
		t.Errorf("ToScreen(_,50).Y = %v, want > ToScreen(_,950).Y = %v (continues past the seam)", wrappedY, nearY)
	}
}

// TestBasicCamera_Visible_ToroidalWrap guards that an entity in the
// wrapped-into-view portion of a toroidal camera is reported visible.
func TestBasicCamera_Visible_ToroidalWrap(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 200, 200), true)
	c.MoveTo(900, 900)

	if !c.Visible(testViewport(30, 30, 30, 30)) {
		t.Error("Visible((30,30)-(60,60)) = false, want true (within the wrapped-into-view portion)")
	}
}

// TestBasicCamera_Visible_PartialViewportDoesNotWrapAtWorldSize guards
// the reported bug: with a viewport far smaller than the world, an
// entity that merely straddles the camera's own reference point (not
// the world's actual 0/size wrap seam) was being canonicalized modulo
// the WORLD's size — sending it almost a full world-length away from
// the reference and making it (wrongly) report as not visible, even
// though it plainly overlaps the visible window's near edge.
func TestBasicCamera_Visible_PartialViewportDoesNotWrapAtWorldSize(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](2000, 2000)
	c := newBasicCamera(surface, testViewport(0, 0, 768, 512), true)
	c.Translate(400, 0) // effective.TopLeft.X is now 400 — nowhere near the world's own edge

	if !c.Visible(testViewport(395, 10, 10, 10)) {
		t.Error("Visible((395,10)-(405,20)) = false, want true (merely straddles the view's near edge)")
	}
}

// TestToScreenQuads_PartialViewportClipsInsteadOfWrapping is the
// ToScreenQuads counterpart: the same straddling entity must render as a
// single quad clipped off the screen's near edge, not split into a
// piece teleported almost a full world-length across the screen.
func TestToScreenQuads_PartialViewportClipsInsteadOfWrapping(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](2000, 2000)
	c := newBasicCamera(surface, testViewport(0, 0, 768, 512), true)
	c.Translate(400, 0)

	quads := c.ToScreenQuads(395, 10, 405, 20)

	if len(quads) != 1 {
		t.Fatalf("len(quads) = %d, want 1 (no wrap split — this isn't near the world's own edge)", len(quads))
	}
	if q := quads[0]; q.X0 != -5 || q.X1 != 5 {
		t.Errorf("quads[0] = %+v, want X0=-5, X1=5 (clipped off the left edge, not teleported to the far side)", q)
	}
}

// TestBasicCamera_FromScreen_ToroidalCanonicalRange guards that FromScreen
// always returns a coordinate in [0, worldSize), matching how entity
// positions are stored, even near the wrap seam.
func TestBasicCamera_FromScreen_ToroidalCanonicalRange(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 200, 200), true)
	c.MoveTo(900, 900)

	x, y := c.FromScreen(150, 150) // deep in the wrapped portion of the view
	if x < 0 || x >= 1000 || y < 0 || y >= 1000 {
		t.Errorf("FromScreen(150,150) = (%v,%v), want both in [0,1000)", x, y)
	}
}

// TestBasicCamera_Bounds_ToroidalReturnsLogicalExtent guards that Bounds
// reports the true (possibly world-exceeding) visible extent for a
// wrapped view, not the world-edge-clamped effective.BottomRight —
// consumers like edge-scroll rely on this to reason about the current
// view without it depending on where the wrap seam currently sits.
func TestBasicCamera_Bounds_ToroidalReturnsLogicalExtent(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](1024, 1024)
	c := newBasicCamera(surface, testViewport(0, 0, 1024, 1024), true)

	c.Translate(100, 0)

	if got := c.Bounds().BottomRight.X; got != 1124 {
		t.Errorf("Bounds().BottomRight.X = %d, want 1124 (100+1024, not clamped to 1024)", got)
	}
}

// TestBasicCamera_ZoomOut_ToroidalCappedToWorldFit guards that a
// toroidal camera's zoom-out is capped exactly like a Euclidean one —
// the renderer only wraps one lap, so zooming out further would show
// emptiness where a second repeat of the world would be needed.
func TestBasicCamera_ZoomOut_ToroidalCappedToWorldFit(t *testing.T) {
	surface := plane.NewToroidal2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(0, 0, 1000, 1000), true)

	c.ZoomOut(10, 500, 500)

	if c.Zoom() < 1 {
		t.Errorf("Zoom() = %v, want >= 1 (viewport already equals world size, one lap max)", c.Zoom())
	}
}

// TestBasicCamera_GobRoundTrip_RefusesToEncode documents why State/Restore
// exist: every basicCamera field is unexported, so gob refuses to encode
// one directly at all — Persistence.Save must go through State() instead.
func TestBasicCamera_GobRoundTrip_RefusesToEncode(t *testing.T) {
	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	c := newBasicCamera(surface, testViewport(10, 10, 100, 100), false)
	c.ZoomIn(2, 60, 60)

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(c); err == nil {
		t.Fatal("Encode of *basicCamera succeeded, want an error (no exported fields)")
	}
}

// TestState_GobRoundTrip_PreservesViewportAndZoom guards the actual
// persistence path: encode/decode a State (not the camera itself), then
// Restore it into a fresh Camera.
func TestState_GobRoundTrip_PreservesViewportAndZoom(t *testing.T) {
	c := NewFromSpace(1000, 1000, false)
	c.MoveTo(50, 60)
	b := c.Bounds()
	cx, cy := float32(b.TopLeft.X+b.BottomRight.X)/2, float32(b.TopLeft.Y+b.BottomRight.Y)/2
	c.ZoomIn(2, cx, cy)
	want := c.State()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(want); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var got State
	if err := gob.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != want {
		t.Fatalf("State after gob round-trip = %+v, want %+v", got, want)
	}

	fresh := NewFromSpace(1000, 1000, false)
	fresh.Restore(got)
	if fresh.Bounds() != c.Bounds() {
		t.Errorf("Bounds() after Restore = %+v, want %+v", fresh.Bounds(), c.Bounds())
	}
	if fresh.Zoom() != c.Zoom() {
		t.Errorf("Zoom() after Restore = %v, want %v", fresh.Zoom(), c.Zoom())
	}
}

func TestNewFromSpace_DefaultViewportMatchesSize(t *testing.T) {
	c := NewFromSpace(800, 600, false)

	bounds := c.Bounds()
	if bounds.TopLeft.X != 0 || bounds.TopLeft.Y != 0 {
		t.Errorf("Bounds().TopLeft = %+v, want (0,0)", bounds.TopLeft)
	}
	if w := bounds.BottomRight.X - bounds.TopLeft.X; w != 800 {
		t.Errorf("Bounds() width = %d, want 800", w)
	}
	if h := bounds.BottomRight.Y - bounds.TopLeft.Y; h != 600 {
		t.Errorf("Bounds() height = %d, want 600", h)
	}
}

func TestNewFromSpace_WithViewportOverride(t *testing.T) {
	override := geom.NewAABBAt(geom.NewVec[uint32](100, 100), 50, 30)
	c := NewFromSpace(800, 600, false, override)

	if got := c.Bounds(); got != override {
		t.Errorf("Bounds() = %+v, want overridden viewport %+v (not the size-derived default)", got, override)
	}
}

func TestNewFromSpace_ToroidalWrapsOnTranslate(t *testing.T) {
	c := NewFromSpace(20, 20, true)

	c.MoveTo(2, 2)
	c.Translate(-10, 0)

	got := c.Bounds()
	if got.TopLeft.X != 12 {
		t.Errorf("after wrap-around Translate, Bounds().TopLeft.X = %d, want 12 (wrapped, not clamped)", got.TopLeft.X)
	}
}

func TestNewFromSpace_EuclideanClampsOnTranslate(t *testing.T) {
	c := NewFromSpace(20, 20, false)

	c.MoveTo(2, 2)
	c.Translate(-10, 0)

	got := c.Bounds()
	if got.TopLeft.X != 0 {
		t.Errorf("Bounds().TopLeft.X = %d, want 0 (clamped euclidean, not wrapped)", got.TopLeft.X)
	}
}

// TestToScreenQuads_NoSplitWhenNotStraddlingReference guards the common
// case: a rectangle that doesn't contain the camera's wrap reference
// point projects as a single, correctly-sized quad.
func TestToScreenQuads_NoSplitWhenNotStraddlingReference(t *testing.T) {
	c := NewFromSpace(1024, 1024, true)
	c.Translate(1000, 0) // effective.TopLeft.X is now 1000 — the wrap reference

	quads := c.ToScreenQuads(1, 0, 11, 10)

	if len(quads) != 1 {
		t.Fatalf("len(quads) = %d, want 1", len(quads))
	}
	q := quads[0]
	if got := q.X1 - q.X0; got != 10 {
		t.Errorf("quad width = %v, want 10", got)
	}
	if got := q.Y1 - q.Y0; got != 10 {
		t.Errorf("quad height = %v, want 10", got)
	}
}

// TestToScreenQuads_SplitsOnSingleAxis guards the reported bug: a
// rectangle straddling a toroidal camera's wrap reference point on one
// axis must split into two screen-space pieces (one wrapped to the far
// edge of the screen) rather than being clipped away entirely.
func TestToScreenQuads_SplitsOnSingleAxis(t *testing.T) {
	c := NewFromSpace(1024, 1024, true)
	c.Translate(1000, 0) // effective.TopLeft.X is now 1000 — the wrap reference

	quads := c.ToScreenQuads(998, 0, 1010, 10)

	if len(quads) != 2 {
		t.Fatalf("len(quads) = %d, want 2", len(quads))
	}
	totalWidth := float32(0)
	for _, q := range quads {
		totalWidth += q.X1 - q.X0
		if got := q.Y1 - q.Y0; got != 10 {
			t.Errorf("quad height = %v, want 10", got)
		}
	}
	if totalWidth != 12 {
		t.Errorf("total quad width straddling the wrap seam = %v, want 12", totalWidth)
	}
	// the piece before the reference (world 998..1000) wraps to the right
	// edge of the screen; the piece after (1000..1010) starts at x=0.
	first, second := quads[0], quads[1]
	if first.X1-first.X0 != 2 || first.T0X != 0 || first.T1X != float32(2)/12 {
		t.Errorf("first quad = %+v, want width 2 spanning T0X=0..%v", first, float32(2)/12)
	}
	if second.X1-second.X0 != 10 || second.X0 != 0 {
		t.Errorf("second quad = %+v, want width 10 starting at screen x=0", second)
	}
}

// TestToScreenQuads_SplitsOnBothAxes guards the corner case: a rectangle
// straddling the reference point on both axes splits into 4 pieces.
func TestToScreenQuads_SplitsOnBothAxes(t *testing.T) {
	c := NewFromSpace(1024, 1024, true)
	c.Translate(1000, 1000) // reference is now (1000,1000)

	quads := c.ToScreenQuads(998, 998, 1010, 1010)

	if len(quads) != 4 {
		t.Fatalf("len(quads) = %d, want 4", len(quads))
	}
	var totalArea float32
	for _, q := range quads {
		totalArea += (q.X1 - q.X0) * (q.Y1 - q.Y0)
	}
	if totalArea != 12*12 {
		t.Errorf("total quad area straddling both wrap seams = %v, want %v", totalArea, 12*12)
	}
}

// TestFromScreenRect_ConsistentAcrossWrapSeam guards that FromScreenRect
// derives the second corner as a scaled world-space offset from the
// first, rather than converting each corner independently through
// FromScreen — which would wrap them to opposite ends of the world when
// the rectangle straddles a toroidal camera's wrap seam.
func TestFromScreenRect_ConsistentAcrossWrapSeam(t *testing.T) {
	c := NewFromSpace(1024, 1024, true)
	c.Translate(1000, 0) // effective.TopLeft.X is now 1000

	// screen x=0 is world x=1000 (the reference); a 20px-wide screen
	// rect here straddles world x=1024≡0, the wrap seam.
	x0, y0, x1, y1 := FromScreenRect(c, 0, 0, 20, 10)

	if got := x1 - x0; got != 20 {
		t.Errorf("FromScreenRect width straddling the wrap seam = %v, want 20", got)
	}
	if y1-y0 != 10 {
		t.Errorf("FromScreenRect height = %v, want 10", y1-y0)
	}
	if x0 != 1000 {
		t.Errorf("FromScreenRect x0 = %v, want 1000", x0)
	}
}

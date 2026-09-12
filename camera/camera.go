package camera

import (
	"math"

	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// AABB is an alias for geom.AABB[uint32], the world-coordinate rectangle type used throughout camera/render.
type AABB = geom.AABB[uint32]

// Quad is one piece of a rectangle projected to screen space by ToScreenQuads; T0X/T1X/T0Y/T1Y give its UV sub-range.
type Quad struct {
	X0, Y0, X1, Y1     float32
	T0X, T1X, T0Y, T1Y float32
}

// FromScreenRect converts a screen-space rectangle's two corners to
// world space the same way, in reverse — the returned x1,y1 may fall
// outside [0, worldSize) for a toroidal camera; pass the result through
// gokg's Space.WrapAABB before querying the spatial index.
func FromScreenRect(cam Camera, sx0, sy0, sx1, sy1 float32) (x0, y0, x1, y1 float32) {
	x0, y0 = cam.FromScreen(sx0, sy0)
	scale := cam.Zoom()
	return x0, y0, x0 + (sx1-sx0)/scale, y0 + (sy1-sy0)/scale
}

// Camera is what renderers, plain click/drag logic, and Runtime need:
// screen conversion, culling, viewport bounds, and control (move/zoom).
type Camera interface {
	ToScreen(x, y float32) (float32, float32)
	// FromScreen inverts ToScreen: screen coordinates back to world coordinates.
	FromScreen(sx, sy float32) (float32, float32)
	// ToScreenQuads projects a rectangle to screen space, splitting it at a toroidal camera's wrap seam when needed.
	ToScreenQuads(x0, y0, x1, y1 float32) []Quad
	Visible(box AABB) bool
	// Bounds returns the current effective (post-zoom) world-space viewport.
	Bounds() AABB
	// MoveTo repositions the visible window's top-left corner, keeping size.
	MoveTo(x, y uint32)
	// Translate shifts the visible window by a signed delta.
	Translate(dx, dy int32)
	// Zoom returns the current zoom factor (1 = default).
	Zoom() float32
	// ZoomIn multiplies the zoom factor by factor, keeping (anchorX, anchorY)
	// (world coordinates) fixed on screen — e.g. the cursor position under FromScreen.
	ZoomIn(factor float32, anchorX, anchorY float32)
	// ZoomOut is ZoomIn(1/factor, anchorX, anchorY).
	ZoomOut(factor float32, anchorX, anchorY float32)
	// State returns the camera's current Viewport/Zoom.
	State() State
	// Persisted returns gob-safe pointers directly into the camera's own live Viewport/Zoom for Persistence.Save/Load to include automatically.
	Persisted() []any
	// Restore rebuilds cached derived state after Persistence.Load decodes directly into Persisted's pointers.
	Restore()
	// SetMinZoom raises ZoomOut's floor above the automatic world-fit
	// one — 0 (the default) applies only that automatic floor.
	SetMinZoom(minZoom float32)
	// SetMaxZoom caps ZoomIn — 0 (the default) leaves zoom-in unrestricted.
	SetMaxZoom(maxZoom float32)
}

// Config optionally overrides a camera's construction — the zero value
// matches NewFromSpace's own defaults (full-surface viewport,
// unrestricted zoom). Unlike Camera's Viewport/Zoom (saved via
// State/Restore), Config is construction-time only and is never persisted.
type Config struct {
	// ViewportWidth/Height size the camera's initial visible window at
	// (0,0) — zero defaults to the full surface.
	ViewportWidth, ViewportHeight uint32
	// MinZoom raises ZoomOut's floor above the automatic world-fit one
	// (which always applies, to prevent showing emptiness beyond the
	// world) — use this to keep gameplay-relevant detail legible even
	// when the world is much larger than the viewport. 0 (default)
	// applies only the automatic floor.
	MinZoom float32
	// MaxZoom caps ZoomIn — 0 (default) leaves zoom-in unrestricted.
	// Useful to stop textures from visibly degrading at extreme close-up.
	MaxZoom float32
}

// State is Camera's persistable visible-window/Zoom snapshot — gob-safe,
// unlike Camera's own concrete implementation (its cache fields are
// unexported and gob would silently drop them).
type State struct {
	Viewport AABB
	Zoom     float32
}

// basicCamera is Camera's only implementation — construct via NewFromSpace.
type basicCamera struct {
	surface      plane.Space2D[uint32]
	viewportSize geom.Vec[uint32] // fixed size at zoom 1 (e.g. screen size)
	toroidal     bool
	minZoomCfg   float32 // 0 = only the automatic world-fit floor applies
	maxZoom      float32 // 0 = unrestricted
	zoom         float32

	effective plane.AABB[uint32] // the current visible window — always valid and clamped to the world
}

var _ Camera = (*basicCamera)(nil)

func newBasicCamera(surface plane.Space2D[uint32], viewport AABB, toroidal bool) *basicCamera {
	w := viewport.BottomRight.X - viewport.TopLeft.X
	h := viewport.BottomRight.Y - viewport.TopLeft.Y
	return &basicCamera{
		surface:      surface,
		viewportSize: geom.NewVec(w, h),
		toroidal:     toroidal,
		zoom:         1,
		effective:    plane.NewAABB(viewport.TopLeft, w, h),
	}
}

// NewFromSpace builds a Camera sized width x height (toroidal or not) — the
// viewport defaults to the full surface at (0,0) unless one is given.
func NewFromSpace(width, height uint32, toroidal bool, viewport ...AABB) Camera {
	var surface plane.Space2D[uint32]
	if toroidal {
		surface = plane.NewToroidal2D(width, height)
	} else {
		surface = plane.NewEuclidean2D(width, height)
	}
	vp := AABB{}
	if len(viewport) > 0 {
		vp = viewport[0]
	}
	if vp.Equals(AABB{}) {
		vp = geom.NewAABBAt(geom.NewVec[uint32](0, 0), width, height)
	}
	return newBasicCamera(surface, vp, toroidal)
}

// NewFromSpaceWithConfig is NewFromSpace plus cfg's viewport size and
// zoom limits — the common case (viewport at the origin, no need for
// NewFromSpace's arbitrary-position override).
func NewFromSpaceWithConfig(width, height uint32, toroidal bool, cfg Config) Camera {
	var viewport []AABB
	if cfg.ViewportWidth != 0 && cfg.ViewportHeight != 0 {
		viewport = []AABB{geom.NewAABBAt(geom.NewVec[uint32](0, 0), cfg.ViewportWidth, cfg.ViewportHeight)}
	}
	cam := NewFromSpace(width, height, toroidal, viewport...)
	if cfg.MinZoom > 0 {
		cam.SetMinZoom(cfg.MinZoom)
	}
	if cfg.MaxZoom > 0 {
		cam.SetMaxZoom(cfg.MaxZoom)
	}
	return cam
}

func wrapRelative(v, ref, size float32) float32 {
	rel := float32(math.Mod(float64(v-ref), float64(size)))
	if rel < 0 {
		rel += size
	}
	return ref + rel
}

func wrapMod(v, size float32) float32 {
	v = float32(math.Mod(float64(v), float64(size)))
	if v < 0 {
		v += size
	}
	return v
}

func windowOffset(x, ref, ww, ws float32) float32 {
	fwd := wrapMod(x-ref, ww)
	distFwd := float32(0)
	if fwd > ws {
		distFwd = fwd - ws
	}
	back := fwd - ww
	distBack := float32(0)
	if back < 0 {
		distBack = -back
	}
	if distBack < distFwd {
		return back
	}
	return fwd
}

func (c *basicCamera) ToScreen(x, y float32) (float32, float32) {
	if c.toroidal {
		world := c.surface.Viewport()
		x = wrapRelative(x, float32(c.effective.TopLeft.X), float32(world.BottomRight.X-world.TopLeft.X))
		y = wrapRelative(y, float32(c.effective.TopLeft.Y), float32(world.BottomRight.Y-world.TopLeft.Y))
	}
	return (x - float32(c.effective.TopLeft.X)) * c.zoom, (y - float32(c.effective.TopLeft.Y)) * c.zoom
}

func (c *basicCamera) ToScreenQuads(x0, y0, x1, y1 float32) []Quad {
	if !c.toroidal {
		sx0, sy0 := c.ToScreen(x0, y0)
		return []Quad{{sx0, sy0, sx0 + (x1-x0)*c.zoom, sy0 + (y1-y0)*c.zoom, 0, 1, 0, 1}}
	}
	world := c.surface.Viewport()
	ww := float32(world.BottomRight.X - world.TopLeft.X)
	wh := float32(world.BottomRight.Y - world.TopLeft.Y)
	refX, refY := float32(c.effective.TopLeft.X), float32(c.effective.TopLeft.Y)
	wsX, wsY := float32(c.effective.Size.X), float32(c.effective.Size.Y)

	u0 := windowOffset(x0, refX, ww, wsX)
	u1 := u0 + (x1 - x0)
	v0 := windowOffset(y0, refY, wh, wsY)
	v1 := v0 + (y1 - y0)

	var quads []Quad
	for _, xp := range splitRange(u0, u1, ww) {
		for _, yp := range splitRange(v0, v1, wh) {
			sx0 := xp.screenLo * c.zoom
			sx1 := sx0 + (xp.hi-xp.lo)*c.zoom
			sy0 := yp.screenLo * c.zoom
			sy1 := sy0 + (yp.hi-yp.lo)*c.zoom
			quads = append(quads, Quad{
				X0: sx0, Y0: sy0, X1: sx1, Y1: sy1,
				T0X: (xp.lo - u0) / (u1 - u0), T1X: (xp.hi - u0) / (u1 - u0),
				T0Y: (yp.lo - v0) / (v1 - v0), T1Y: (yp.hi - v0) / (v1 - v0),
			})
		}
	}
	return quads
}

type rangePiece struct{ lo, hi, screenLo float32 }

func splitRange(u0, u1, size float32) []rangePiece {
	if u1 <= size {
		return []rangePiece{{u0, u1, u0}}
	}
	return []rangePiece{{u0, size, u0}, {size, u1, 0}}
}

func (c *basicCamera) FromScreen(sx, sy float32) (float32, float32) {
	x := sx/c.zoom + float32(c.effective.TopLeft.X)
	y := sy/c.zoom + float32(c.effective.TopLeft.Y)
	if c.toroidal {
		world := c.surface.Viewport()
		x = wrapMod(x, float32(world.BottomRight.X-world.TopLeft.X))
		y = wrapMod(y, float32(world.BottomRight.Y-world.TopLeft.Y))
	}
	return x, y
}

// Visible uses strict >/< — touching edges are not visible.
func (c *basicCamera) Visible(box AABB) bool {
	tlX, tlY := float32(box.TopLeft.X), float32(box.TopLeft.Y)
	brX, brY := float32(box.BottomRight.X), float32(box.BottomRight.Y)
	if c.toroidal {
		world := c.surface.Viewport()
		ww := float32(world.BottomRight.X - world.TopLeft.X)
		wh := float32(world.BottomRight.Y - world.TopLeft.Y)
		refX, refY := float32(c.effective.TopLeft.X), float32(c.effective.TopLeft.Y)
		wsX, wsY := float32(c.effective.Size.X), float32(c.effective.Size.Y)

		width, height := brX-tlX, brY-tlY
		tlX = refX + windowOffset(tlX, refX, ww, wsX)
		brX = tlX + width
		tlY = refY + windowOffset(tlY, refY, wh, wsY)
		brY = tlY + height
	}
	far := c.Bounds()
	return brX > float32(c.effective.TopLeft.X) && tlX < float32(far.BottomRight.X) &&
		brY > float32(c.effective.TopLeft.Y) && tlY < float32(far.BottomRight.Y)
}

// Bounds returns the logical (unclamped) visible extent — for a
// wrapped-around toroidal view this may exceed the world's own size,
// unlike effective.BottomRight (clamped to the world's edge).
func (c *basicCamera) Bounds() AABB {
	return AABB{
		TopLeft: c.effective.TopLeft,
		BottomRight: geom.NewVec(
			c.effective.TopLeft.X+c.effective.Size.X,
			c.effective.TopLeft.Y+c.effective.Size.Y,
		),
	}
}

// MoveTo repositions the visible window's top-left corner, keeping size.
func (c *basicCamera) MoveTo(x, y uint32) {
	dx := int32(int64(x) - int64(c.effective.TopLeft.X))
	dy := int32(int64(y) - int64(c.effective.TopLeft.Y))
	c.Translate(dx, dy)
}

// Translate shifts the visible window by a signed delta, clamping
// (Euclidean) or wrapping (Toroidal) it against the world via gokg's
// Reposition — effective is the single source of truth for position, so
// this never needs to reason about a separately-tracked reference box.
func (c *basicCamera) Translate(dx, dy int32) {
	c.surface.Reposition(&c.effective, geom.NewVec(uint32(dx), uint32(dy)))
}

// Zoom returns the current zoom factor (1 = default).
func (c *basicCamera) Zoom() float32 { return c.zoom }

// ZoomIn multiplies the zoom factor by factor, keeping (anchorX, anchorY)
// (world coordinates) fixed on screen — e.g. the cursor position under FromScreen.
func (c *basicCamera) ZoomIn(factor float32, anchorX, anchorY float32) {
	beforeX, beforeY := c.ToScreen(anchorX, anchorY)

	newZoom := c.zoom * factor
	if newZoom < 0.01 {
		newZoom = 0.01
	}
	if min := c.minZoom(); newZoom < min {
		newZoom = min
	}
	if c.maxZoom > 0 && newZoom > c.maxZoom {
		newZoom = c.maxZoom
	}
	c.setZoom(newZoom)

	afterX, afterY := c.ToScreen(anchorX, anchorY)
	dx := int32((afterX - beforeX) / c.zoom)
	dy := int32((afterY - beforeY) / c.zoom)
	if dx != 0 || dy != 0 {
		c.Translate(dx, dy)
	}
}

// setZoom resizes effective for the new zoom, keeping its center fixed
// as much as possible, then clamps it back into the world via Reposition
// — the same mechanism Translate uses, so a center that would need a
// negative top-left clamps correctly instead of underflowing.
func (c *basicCamera) setZoom(zoom float32) {
	cx := int64(c.effective.TopLeft.X) + int64(c.effective.Size.X)/2
	cy := int64(c.effective.TopLeft.Y) + int64(c.effective.Size.Y)/2

	w := uint32(float64(c.viewportSize.X) / float64(zoom))
	h := uint32(float64(c.viewportSize.Y) / float64(zoom))

	desiredX := cx - int64(w)/2
	desiredY := cy - int64(h)/2
	deltaX := int32(desiredX - int64(c.effective.TopLeft.X))
	deltaY := int32(desiredY - int64(c.effective.TopLeft.Y))

	eff := plane.NewAABB(c.effective.TopLeft, w, h)
	c.surface.Reposition(&eff, geom.NewVec(uint32(deltaX), uint32(deltaY)))

	c.zoom = zoom
	c.effective = eff
}

// minZoom returns the smallest zoom ZoomIn/ZoomOut will settle at — the
// larger of the automatic world-fit floor (never reveal emptiness beyond
// the world) and any configured SetMinZoom override.
func (c *basicCamera) minZoom() float32 {
	world := c.surface.Viewport()
	worldW := float32(world.BottomRight.X - world.TopLeft.X)
	worldH := float32(world.BottomRight.Y - world.TopLeft.Y)
	byW := float32(c.viewportSize.X) / worldW
	byH := float32(c.viewportSize.Y) / worldH
	floor := byW
	if byH > floor {
		floor = byH
	}
	if c.minZoomCfg > floor {
		floor = c.minZoomCfg
	}
	return floor
}

// SetMinZoom raises ZoomOut's floor above the automatic world-fit one.
func (c *basicCamera) SetMinZoom(minZoom float32) { c.minZoomCfg = minZoom }

// SetMaxZoom caps ZoomIn — 0 (the default) leaves zoom-in unrestricted.
func (c *basicCamera) SetMaxZoom(maxZoom float32) { c.maxZoom = maxZoom }

// ZoomOut is ZoomIn(1/factor, anchorX, anchorY).
func (c *basicCamera) ZoomOut(factor float32, anchorX, anchorY float32) {
	c.ZoomIn(1/factor, anchorX, anchorY)
}

// State returns the camera's current Viewport/Zoom.
func (c *basicCamera) State() State { return State{Viewport: c.effective.AABB, Zoom: c.zoom} }

// Persisted returns gob-safe pointers directly into the camera's own live Viewport/Zoom for Persistence.Save/Load to include automatically.
func (c *basicCamera) Persisted() []any {
	return []any{&c.effective.AABB, &c.zoom}
}

// Restore rebuilds effective's cached derived state after Persistence.Load decodes directly into Persisted's pointers.
func (c *basicCamera) Restore() {
	w := c.effective.BottomRight.X - c.effective.TopLeft.X
	h := c.effective.BottomRight.Y - c.effective.TopLeft.Y
	c.effective = plane.NewAABB(c.effective.TopLeft, w, h)
}

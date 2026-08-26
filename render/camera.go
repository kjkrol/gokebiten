package render

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// AABB is an alias for geom.AABB[uint32], the world-coordinate rectangle type used throughout render.
type AABB = geom.AABB[uint32]

// Camera is what renderers need: screen conversion, culling, viewport bounds. Control (move/zoom) is not part of it.
type Camera interface {
	ToScreen(x, y float32) (float32, float32)
	Visible(box AABB) bool
	// Bounds returns the current effective (post-zoom) world-space viewport.
	Bounds() AABB
	// ExtendedBounds returns Bounds expanded by margin on every side.
	ExtendedBounds(margin uint32) AABB
}

// BasicCamera is the default Camera, wrapped/clamped by a plane.Space2D matching the world's topology.
// Never pass *BasicCamera to Game.Save/Load — gob silently drops its unexported fields; save Viewport/Zoom() instead.
type BasicCamera struct {
	surface  plane.Space2D[uint32]
	Viewport plane.AABB[uint32]
	zoom     float32

	effective plane.AABB[uint32]
	scale     float32
}

// NewBasicCamera builds a camera over surface at zoom 1.
func NewBasicCamera(surface plane.Space2D[uint32], viewport AABB) *BasicCamera {
	w := viewport.BottomRight.X - viewport.TopLeft.X
	h := viewport.BottomRight.Y - viewport.TopLeft.Y
	c := &BasicCamera{surface: surface, Viewport: plane.NewAABB(viewport.TopLeft, w, h), zoom: 1}
	c.recompute()
	return c
}

// recompute refreshes the effective/scale cache from Viewport+zoom.
func (c *BasicCamera) recompute() {
	if c.zoom == 1 {
		c.effective = c.Viewport
		c.scale = 1
		return
	}
	w := int64(c.Viewport.BottomRight.X) - int64(c.Viewport.TopLeft.X)
	h := int64(c.Viewport.BottomRight.Y) - int64(c.Viewport.TopLeft.Y)
	cx := int64(c.Viewport.TopLeft.X) + w/2
	cy := int64(c.Viewport.TopLeft.Y) + h/2
	hw := int64(float64(w) / 2 / float64(c.zoom))
	hh := int64(float64(h) / 2 / float64(c.zoom))
	eff := plane.NewAABB(geom.NewVec(uint32(cx-hw), uint32(cy-hh)), uint32(2*hw), uint32(2*hh))
	c.surface.Normalize(eff.AABB)
	c.effective = eff
	c.scale = c.zoom
}

func (c *BasicCamera) ToScreen(x, y float32) (float32, float32) {
	return (x - float32(c.effective.TopLeft.X)) * c.scale, (y - float32(c.effective.TopLeft.Y)) * c.scale
}

// Visible uses strict >/< — touching edges are not visible.
func (c *BasicCamera) Visible(box AABB) bool {
	return box.BottomRight.X > c.effective.TopLeft.X && box.TopLeft.X < c.effective.BottomRight.X &&
		box.BottomRight.Y > c.effective.TopLeft.Y && box.TopLeft.Y < c.effective.BottomRight.Y
}

func (c *BasicCamera) Bounds() AABB { return c.effective.AABB }

// ExtendedBounds grows Bounds by margin on every side via surface.Expand.
func (c *BasicCamera) ExtendedBounds(margin uint32) AABB {
	expanded := c.effective
	c.surface.Expand(&expanded, margin)
	return expanded.AABB
}

// MoveTo repositions the reference top-left corner, keeping size.
func (c *BasicCamera) MoveTo(x, y uint32) {
	dx := int32(int64(x) - int64(c.Viewport.TopLeft.X))
	dy := int32(int64(y) - int64(c.Viewport.TopLeft.Y))
	c.Translate(dx, dy)
}

// Translate shifts the reference viewport by a signed delta.
func (c *BasicCamera) Translate(dx, dy int32) {
	c.surface.Translate(&c.Viewport, geom.NewVec(uint32(dx), uint32(dy)))
	c.recompute()
}

// Zoom returns the current zoom factor (1 = default).
func (c *BasicCamera) Zoom() float32 { return c.zoom }

// ZoomIn multiplies the zoom factor by factor, anchored on the viewport's center.
func (c *BasicCamera) ZoomIn(factor float32) {
	c.zoom *= factor
	if c.zoom < 0.01 {
		c.zoom = 0.01
	}
	c.recompute()
}

// ZoomOut is ZoomIn(1/factor).
func (c *BasicCamera) ZoomOut(factor float32) { c.ZoomIn(1 / factor) }

var _ Camera = (*BasicCamera)(nil)

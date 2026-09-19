package render

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// VisitWrapImages calls fn once per image of box — the canonical one first,
// then each toroidal fragment — handing over that image's own box together with
// the world-space shift at which it sits. A box in a Euclidean space, or one
// that reaches no edge, has a single image and no shift.
//
// plane.AABB carries the fragments; this is the one place that says what they
// mean for drawing, so everything wrapping a shape at the world seam agrees
// about it. A fragment spanning the parent's right edge reappears at the left,
// one world width back, which is why the shift is negative.
//
// What each caller takes from an image differs: something textured draws the
// image's own box and picks the matching slice of its texture, while a shape
// that cannot be sliced is drawn whole at the shift instead. fn returning false
// stops the walk.
func VisitWrapImages(
	box plane.AABB,
	worldW, worldH float32,
	fn func(img geom.AABB, dx, dy float32) bool,
) {
	if !fn(box.AABB, 0, 0) {
		return
	}
	box.VisitFragments(func(fp plane.FragPosition, frag geom.AABB) bool {
		var dx, dy float32
		if fp == plane.FRAG_RIGHT || fp == plane.FRAG_BOTTOM_RIGHT {
			dx = -worldW
		}
		if fp == plane.FRAG_BOTTOM || fp == plane.FRAG_BOTTOM_RIGHT {
			dy = -worldH
		}
		return fn(frag, dx, dy)
	})
}

package render

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
)

// VisitWrapImages calls fn with the box and world shift of each image of box, until fn says stop.
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

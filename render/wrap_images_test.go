package render_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

type image struct {
	box    geom.AABB
	dx, dy float32
}

func images(box plane.AABB, w, h float32) []image {
	var out []image
	render.VisitWrapImages(box, w, h, func(b geom.AABB, dx, dy float32) bool {
		out = append(out, image{b, dx, dy})
		return true
	})
	return out
}

func wrapped(toroidal bool, x, y, w, h float64) plane.AABB {
	var space plane.Space2D
	if toroidal {
		space = plane.NewToroidal2D(1000, 1000)
	} else {
		space = plane.NewEuclidean2D(1000, 1000)
	}
	return space.WrapAABB(geom.NewAABBAt(geom.NewVec(x, y), w, h))
}

func TestVisitWrapImages_EuclideanBoxHasOneImage(t *testing.T) {
	got := images(wrapped(false, 970, 960, 100, 100), 1000, 1000)
	if len(got) != 1 {
		t.Fatalf("%d images in a Euclidean space, want 1", len(got))
	}
	if got[0].dx != 0 || got[0].dy != 0 {
		t.Errorf("shift (%v,%v), want none", got[0].dx, got[0].dy)
	}
}

func TestVisitWrapImages_BoxInsideTheWorldHasOneImage(t *testing.T) {
	if got := images(wrapped(true, 100, 100, 50, 50), 1000, 1000); len(got) != 1 {
		t.Fatalf("%d images for a box reaching no edge, want 1", len(got))
	}
}

func TestVisitWrapImages_BoxOnOneEdgeHasTwo(t *testing.T) {
	right := images(wrapped(true, 970, 100, 100, 50), 1000, 1000)
	if len(right) != 2 {
		t.Fatalf("%d images across the right edge, want 2", len(right))
	}
	if right[1].dx != -1000 || right[1].dy != 0 {
		t.Errorf("wrapped copy sits at (%v,%v), want (-1000,0)", right[1].dx, right[1].dy)
	}

	bottom := images(wrapped(true, 100, 960, 50, 100), 1000, 1000)
	if len(bottom) != 2 {
		t.Fatalf("%d images across the bottom edge, want 2", len(bottom))
	}
	if bottom[1].dx != 0 || bottom[1].dy != -1000 {
		t.Errorf("wrapped copy sits at (%v,%v), want (0,-1000)", bottom[1].dx, bottom[1].dy)
	}
}

func TestVisitWrapImages_BoxInTheCornerHasFour(t *testing.T) {
	got := images(wrapped(true, 970, 960, 100, 100), 1000, 1000)
	if len(got) != 4 {
		t.Fatalf("%d images in the world's corner, want 4", len(got))
	}
	want := [][2]float32{{0, 0}, {-1000, 0}, {0, -1000}, {-1000, -1000}}
	for i, w := range want {
		if got[i].dx != w[0] || got[i].dy != w[1] {
			t.Errorf("image %d sits at (%v,%v), want (%v,%v)", i, got[i].dx, got[i].dy, w[0], w[1])
		}
	}
}

// The canonical image comes first, so a caller that only wants the main one can
// stop after it.
func TestVisitWrapImages_StopsWhenAsked(t *testing.T) {
	seen := 0
	render.VisitWrapImages(wrapped(true, 970, 960, 100, 100), 1000, 1000,
		func(geom.AABB, float32, float32) bool {
			seen++
			return false
		})
	if seen != 1 {
		t.Errorf("visited %d images after returning false, want 1", seen)
	}
}

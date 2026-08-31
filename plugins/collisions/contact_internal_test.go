package collisions

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

func TestAbs32(t *testing.T) {
	cases := []struct {
		in, want int32
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{1, 1},
		{-1, 1},
	}
	for _, c := range cases {
		if got := abs32(c.in); got != c.want {
			t.Errorf("abs32(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func aabbAt(x, y, w, h uint32) geom.AABB[uint32] {
	return geom.NewAABB(geom.NewVec(x, y), geom.NewVec(x+w, y+h))
}

func TestContact_Penetration(t *testing.T) {
	var c Contact

	t.Run("no overlap", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(100, 100, 10, 10)
		pen := c.penetration(r1, r2)
		if pen.X != 0 || pen.Y != 0 {
			t.Errorf("penetration = %+v, want zero", pen)
		}
	})

	t.Run("touching but not overlapping", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(10, 0, 10, 10) // r1's right edge == r2's left edge
		pen := c.penetration(r1, r2)
		if pen.X != 0 || pen.Y != 0 {
			t.Errorf("penetration = %+v, want zero (edges touching only)", pen)
		}
	})

	t.Run("small overlap on X pushes r1 left, out its near edge", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(8, 0, 10, 10) // 2px overlap on X, full overlap on Y
		pen := c.penetration(r1, r2)
		if pen.X != -2 || pen.Y != 0 {
			t.Errorf("penetration = %+v, want {X:-2 Y:0}", pen)
		}
	})

	t.Run("small overlap on Y pushes r1 up, out its near edge", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(0, 9, 10, 10) // 1px overlap on Y, full overlap on X
		pen := c.penetration(r1, r2)
		if pen.X != 0 || pen.Y != -1 {
			t.Errorf("penetration = %+v, want {X:0 Y:-1}", pen)
		}
	})

	t.Run("r2 fully inside r1 picks the smallest push direction", func(t *testing.T) {
		r1 := aabbAt(0, 0, 100, 100)
		r2 := aabbAt(90, 40, 5, 5) // closest edge is r1's right edge, 15px away
		pen := c.penetration(r1, r2)
		if pen.X != -10 || pen.Y != 0 {
			t.Errorf("penetration = %+v, want {X:-10 Y:0} (push r1 left, out r1's near edge)", pen)
		}
	})
}

func TestContact_CalculateMtv(t *testing.T) {
	var c Contact

	t.Run("no penetration returns res=false", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(100, 100, 10, 10)
		mtv1, mtv2, res := c.calculateMtv(r1, r2, false)
		if res {
			t.Fatalf("expected res=false for non-overlapping boxes, got mtv1=%+v mtv2=%+v", mtv1, mtv2)
		}
	})

	t.Run("dynamic-dynamic even penetration splits evenly, opposite directions", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(6, 0, 10, 10) // 4px overlap on X -> penetration.X == -4
		mtv1, mtv2, res := c.calculateMtv(r1, r2, false)
		if !res {
			t.Fatal("expected res=true")
		}
		if int32(mtv1.X) != -2 || int32(mtv2.X) != 2 {
			t.Errorf("mtv1.X=%d mtv2.X=%d, want -2/2 (even 4px split, pushed apart)", int32(mtv1.X), int32(mtv2.X))
		}
	})

	t.Run("dynamic-dynamic odd penetration loses no pixel", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(7, 0, 10, 10) // 3px overlap on X -> penetration.X == -3
		mtv1, mtv2, res := c.calculateMtv(r1, r2, false)
		if !res {
			t.Fatal("expected res=true")
		}
		if got := abs32(int32(mtv1.X)) + abs32(int32(mtv2.X)); got != 3 {
			t.Errorf("|mtv1.X|+|mtv2.X| = %d, want 3 (no pixel lost splitting an odd penetration)", got)
		}
	})

	t.Run("static B pushes all penetration onto A", func(t *testing.T) {
		r1 := aabbAt(0, 0, 10, 10)
		r2 := aabbAt(8, 0, 10, 10) // 2px overlap on X -> penetration.X == -2
		mtv1, mtv2, res := c.calculateMtv(r1, r2, true)
		if !res {
			t.Fatal("expected res=true")
		}
		if int32(mtv1.X) != -2 || mtv2.X != 0 {
			t.Errorf("mtv1=%+v mtv2=%+v, want all push on A (mtv1.X=-2, mtv2.X=0) for a static B", mtv1, mtv2)
		}
	})
}

func newPos(x, y, w, h uint32) *world.Position {
	return &world.Position{AABB: plane.NewAABB(geom.NewVec(x, y), w, h)}
}

func TestContact_FindActiveCollision(t *testing.T) {
	t.Run("no fragments uses main boxes", func(t *testing.T) {
		c := &Contact{
			A: contactSide{Pos: newPos(0, 0, 10, 10)},
			B: contactSide{Pos: newPos(8, 0, 10, 10)},
		}
		boxA, boxB, pen := c.findActiveCollision()
		if pen.X != -2 || pen.Y != 0 {
			t.Errorf("pen = %+v, want {X:-2 Y:0}", pen)
		}
		if !boxA.Equals(c.A.Pos.AABB.AABB) || !boxB.Equals(c.B.Pos.AABB.AABB) {
			t.Errorf("boxA/boxB should equal the main AABBs when neither side has fragments")
		}
	})

	t.Run("no overlap anywhere returns zero pen", func(t *testing.T) {
		c := &Contact{
			A: contactSide{Pos: newPos(0, 0, 10, 10)},
			B: contactSide{Pos: newPos(1000, 1000, 10, 10)},
		}
		_, _, pen := c.findActiveCollision()
		if pen.X != 0 || pen.Y != 0 {
			t.Errorf("pen = %+v, want zero for non-overlapping main boxes", pen)
		}
	})

	t.Run("stronger overlap on B's fragment wins over the main combo", func(t *testing.T) {
		posB := newPos(8, 0, 10, 10)
		// A weak 1px overlap on the main box, a much stronger 5px overlap on
		// a synthetic fragment placed to overlap posA more.
		posB.AABB.Frags[0] = aabbAt(3, 0, 10, 10) // FRAG_RIGHT slot; VisitFragments needs FragMask set
		posB.AABB.FragMask |= 1 << 1              // FRAG_RIGHT = 1

		c := &Contact{
			A: contactSide{Pos: newPos(0, 0, 10, 10)},
			B: contactSide{Pos: posB},
		}
		boxA, boxB, pen := c.findActiveCollision()
		if pen.X != -7 {
			t.Errorf("pen.X = %d, want -7 (the fragment's overlap should have won)", pen.X)
		}
		if !boxA.Equals(c.A.Pos.AABB.AABB) {
			t.Errorf("boxA should stay A's main box")
		}
		if !boxB.Equals(posB.AABB.Frags[0]) {
			t.Errorf("boxB should be B's winning fragment, got %+v want %+v", boxB, posB.AABB.Frags[0])
		}
	})

	t.Run("A's fragment finds an overlap the main boxes don't have", func(t *testing.T) {
		posA := newPos(0, 0, 10, 10)
		posA.AABB.Frags[0] = aabbAt(5, 0, 10, 10) // FRAG_RIGHT slot
		posA.AABB.FragMask |= 1 << 1

		c := &Contact{
			A: contactSide{Pos: posA},
			B: contactSide{Pos: newPos(12, 0, 10, 10)}, // no overlap with A's main box at all
		}
		boxA, boxB, pen := c.findActiveCollision()
		if pen.X != -3 {
			t.Errorf("pen.X = %d, want -3 (A's fragment is the only combo that overlaps)", pen.X)
		}
		if !boxA.Equals(posA.AABB.Frags[0]) {
			t.Errorf("boxA should be A's winning fragment, got %+v want %+v", boxA, posA.AABB.Frags[0])
		}
		if !boxB.Equals(c.B.Pos.AABB.AABB) {
			t.Errorf("boxB should stay B's main box")
		}
	})
}

func TestHasAnyFragment(t *testing.T) {
	ab := plane.NewAABB(geom.NewVec[uint32](0, 0), 10, 10)
	if hasAnyFragment(&ab) {
		t.Error("expected a fresh AABB to have no fragments")
	}

	ab.Frags[0] = aabbAt(0, 0, 1, 1)
	ab.FragMask |= 1 << 1
	if !hasAnyFragment(&ab) {
		t.Error("expected hasAnyFragment to be true once FragMask has a bit set")
	}
}

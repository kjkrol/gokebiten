package board

import (
	"math"
	"testing"

	"github.com/kjkrol/aabbworld/geom"
)

func inAny(p geom.Vec, boxes []geom.AABB) bool {
	for _, b := range boxes {
		if p.X >= b.TopLeft.X-1e-9 && p.X <= b.BottomRight.X+1e-9 && p.Y >= b.TopLeft.Y-1e-9 && p.Y <= b.BottomRight.Y+1e-9 {
			return true
		}
	}
	return false
}

func TestHexGrid_CellBoxesCoverEveryVertexWithinTheBoundingBox(t *testing.T) {
	g := newHexGrid(3, 3, 20)
	c := packAxial(1, 1)
	center := g.CellCenter(c)
	boxes := g.CellBoxes(c, nil)
	if len(boxes) != 2*HexCapStrips+1 {
		t.Fatalf("%d boxes, want %d", len(boxes), 2*HexCapStrips+1)
	}
	halfW := math.Sqrt(3) / 2 * g.Size
	for i := range 6 {
		a := math.Pi/6 + float64(i)*math.Pi/3 // pointy-top vertices
		v := geom.NewVec(center.X+g.Size*math.Cos(a), center.Y+g.Size*math.Sin(a))
		if !inAny(v, boxes) {
			t.Errorf("vertex %d at %v lies outside every box", i, v)
		}
	}
	for _, b := range boxes {
		if b.TopLeft.X < center.X-halfW-1e-9 || b.BottomRight.X > center.X+halfW+1e-9 ||
			b.TopLeft.Y < center.Y-g.Size-1e-9 || b.BottomRight.Y > center.Y+g.Size+1e-9 {
			t.Errorf("box %v sticks out of the hex's bounding box", b)
		}
	}
}

func wallBoard(w, h uint32, cells func(x, y uint32) bool) *Board {
	brd := NewBoard(newSquareGrid(w, h, 10), NewTerrainMap())
	brd.SetAll(CellKind{Name: "grass", Allows: Land})
	wall := CellKind{Name: "wall", Solid: true}
	for y := range h {
		for x := range w {
			if cells(x, y) {
				c, _ := brd.CellIndex(x, y)
				brd.Set(c, wall)
			}
		}
	}
	return brd
}

func TestTerrainBoxes_MergesAColumnIntoOneBody(t *testing.T) {
	brd := wallBoard(6, 16, func(x, y uint32) bool { return x == 3 && y >= 1 && y <= 14 })
	got := terrainBoxes(brd, nil)
	if len(got) != 1 {
		t.Fatalf("%d bodies, want 1: %+v", len(got), got)
	}
	want := geom.NewAABB(geom.NewVec(30, 10), geom.NewVec(40, 150))
	if !got[0].box.Equals(want) {
		t.Errorf("body %v, want %v", got[0].box, want)
	}
}

func TestTerrainBoxes_CapsABodyAtMaxBodyCells(t *testing.T) {
	brd := wallBoard(3, 20, func(x, y uint32) bool { return true })
	got := terrainBoxes(brd, nil)
	if len(got) != 2 {
		t.Fatalf("%d bodies for a 3x20 block, want 2", len(got))
	}
	if h := got[0].box.BottomRight.Y - got[0].box.TopLeft.Y; h != MaxBodyCells*10 {
		t.Errorf("first body is %v tall, want %d", h, MaxBodyCells*10)
	}
}

func TestTerrainBoxes_DefaultKindCountsWhenImpassable(t *testing.T) {
	brd := NewBoard(newSquareGrid(4, 1, 10), NewTerrainMap())
	brd.SetAll(CellKind{Name: "rock", Solid: true})
	c, _ := brd.CellIndex(1, 0)
	brd.Set(c, CellKind{Name: "grass", Allows: Land})
	got := terrainBoxes(brd, nil)
	if len(got) != 2 {
		t.Fatalf("%d bodies, want the rock either side of the grass", len(got))
	}
}

func TestTerrainBoxes_NeverJoinsAcrossTheWrapSeamOrAcrossKinds(t *testing.T) {
	brd := wallBoard(5, 1, func(x, y uint32) bool { return x == 0 || x == 4 })
	brd.Grid.(*squareGrid).WrapX = true
	if got := terrainBoxes(brd, nil); len(got) != 2 {
		t.Errorf("%d bodies on a wrapping row, want 2 either side of the seam", len(got))
	}

	brd = wallBoard(2, 1, func(x, y uint32) bool { return true })
	c, _ := brd.CellIndex(1, 0)
	brd.Set(c, CellKind{Name: "water", Solid: true})
	if got := terrainBoxes(brd, nil); len(got) != 2 {
		t.Errorf("%d bodies of two kinds, want 2", len(got))
	}
}

func TestHexGrid_CellOutlineAndBounds(t *testing.T) {
	g := newHexGrid(3, 3, 20)
	c := packAxial(1, 1)
	center := g.CellCenter(c)
	pts := g.CellOutline(c, nil)
	if len(pts) != 6 {
		t.Fatalf("%d corners, want 6", len(pts))
	}
	for i, p := range pts {
		if d := math.Hypot(p.X-center.X, p.Y-center.Y); math.Abs(d-g.Size) > 1e-9 {
			t.Errorf("corner %d is %v from the centre, want %v", i, d, g.Size)
		}
	}
	if pts[0].Y >= center.Y || math.Abs(pts[0].X-center.X) > 1e-9 {
		t.Errorf("first corner %v is not the top point of a pointy-top hex", pts[0])
	}
	w, h := g.CellBounds()
	if math.Abs(w-math.Sqrt(3)*20) > 1e-9 || h != 40 {
		t.Errorf("bounds %v×%v, want √3·20×40", w, h)
	}
	sw, sh := newSquareGrid(2, 2, 16).CellBounds()
	if sw != 16 || sh != 16 {
		t.Errorf("square bounds %v×%v, want 16×16", sw, sh)
	}
}

func TestTerrainBoxes_VeiledCellsAreBodiesButNotSolid(t *testing.T) {
	brd := wallBoard(3, 1, func(x, y uint32) bool { return x == 0 })
	c, _ := brd.CellIndex(2, 0)
	brd.Set(c, CellKind{Name: "forest", Allows: Land, Veil: 0.6})
	got := terrainBoxes(brd, nil)
	if len(got) != 2 {
		t.Fatalf("%d bodies, want a wall and a forest", len(got))
	}
	solid := map[string]bool{}
	for _, b := range got {
		solid[b.kind] = b.solid
	}
	if !solid["wall"] || solid["forest"] {
		t.Errorf("solid by kind = %v, want wall solid and forest not", solid)
	}
	for _, b := range got {
		if b.kind == "forest" && b.veil != 0.6 {
			t.Errorf("forest body veil = %v, want 0.6", b.veil)
		}
	}
}

func TestCellsUnder_SquareAndHex(t *testing.T) {
	sq := newSquareGrid(4, 4, 10)
	count := func(g Grid, box geom.AABB) (n int) { g.CellsUnder(box, func(CellID) { n++ }); return }
	if n := count(sq, geom.NewAABB(geom.NewVec(2, 2), geom.NewVec(8, 8))); n != 1 {
		t.Errorf("a box inside one square touches %d cells, want 1", n)
	}
	if n := count(sq, geom.NewAABB(geom.NewVec(8, 8), geom.NewVec(12, 12))); n != 4 {
		t.Errorf("a box over a corner of four squares touches %d cells, want 4", n)
	}
	if n := count(sq, geom.NewAABB(geom.NewVec(1, 1), geom.NewVec(39, 9))); n != 4 {
		t.Errorf("a box along the top row touches %d cells, want 4", n)
	}

	hx := newHexGrid(3, 3, 20)
	c := packAxial(1, 1)
	center := hx.CellCenter(c)
	tip := geom.NewAABB(geom.NewVec(center.X-1, center.Y-hx.Size+0.5), geom.NewVec(center.X+1, center.Y-hx.Size+2))
	found := false
	hx.CellsUnder(tip, func(got CellID) { found = found || got == c })
	if !found {
		t.Error("a box on the very tip of a hex does not touch it")
	}
	beside := geom.NewAABB(geom.NewVec(center.X+hx.Size, center.Y-1), geom.NewVec(center.X+hx.Size+2, center.Y+1))
	hx.CellsUnder(beside, func(got CellID) {
		if got == c {
			t.Error("a box past the hex's side touches it")
		}
	})
}

func TestTerrainMap_VersionMovesOnlyWhenTheKindChanges(t *testing.T) {
	tm := NewTerrainMap()
	grass := CellKind{Name: "grass", Allows: Land}
	before := tm.Version()
	tm.Set(1, grass)
	tm.Set(1, grass)
	tm.SetMany([]CellID{1, 1}, grass)
	if got := tm.Version() - before; got != 1 {
		t.Errorf("version moved %d times for one real change, want 1", got)
	}
	tm.SetMany([]CellID{1, 2}, CellKind{Name: "snow", Allows: Land})
	if got := tm.Version() - before; got != 2 {
		t.Errorf("version moved %d times after a second change, want 2", got)
	}
}

func TestCellKind_CostForPicksTheCheapestOfTheEntitysAdmittedDomains(t *testing.T) {
	const sylvan = Domain(1 << 3)
	forest := CellKind{Name: "forest", Cost: 3, Allows: Land | sylvan}.Costing(sylvan, 1)
	for _, tc := range []struct {
		d    Domain
		want float64
	}{
		{Land, 3}, {sylvan, 1}, {Land | sylvan, 1}, {Water, 3}, {Water | sylvan, 1},
	} {
		if got := forest.CostFor(tc.d); got != tc.want {
			t.Errorf("CostFor(%08b) = %v, want %v", tc.d, got, tc.want)
		}
	}
	snow := CellKind{Name: "snow", Cost: 3, Allows: Land}.Costing(Air, 1) // priced for Air, but not admitted
	if got := snow.CostFor(Air); got != 3 {
		t.Errorf("a domain the kind does not admit pays %v, want the plain Cost 3", got)
	}
}

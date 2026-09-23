package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/render"
)

// flatAtlas is an AtlasSource with no sheet: enough for gathering quads without drawing.
type flatAtlas struct{}

func (flatAtlas) Atlas() *ebiten.Image                            { return nil }
func (flatAtlas) UV(render.SpriteID) (sx0, sy0, sx1, sy1 float32) { return 0, 0, 1, 1 }

// countingModifier counts the entities the renderer resolved layers for.
type countingModifier struct{ n int }

func (*countingModifier) Bind(*goke.QueryBuilder) {}
func (m *countingModifier) Apply(_ *goke.Cursor, _ int, _ *Base, acc []Appearance) []Appearance {
	m.n++
	return acc
}

// drawScene puts one 10x10 entity at each of at into a world of the given edges and draws it
// once through cam, returning how many entities had their layers resolved.
func drawScene(t *testing.T, edges aabbworld.Edges, cam camera.Camera, at ...geom.Vec) int {
	t.Helper()
	space, err := aabbworld.NewSpace(aabbworld.Config{Width: 1000, Height: 1000, Edges: edges, BucketSize: 64})
	if err != nil {
		t.Fatal(err)
	}
	r := newRenderer(cam, flatAtlas{}, space, 1000, 1000)
	counter := &countingModifier{}
	r.WithModifier(counter)

	var base goke.Comp[Base]
	var appearance goke.Comp[Appearance]
	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&base, &appearance)
		f.Create(len(at))
		var items []aabbworld.Item
		i := 0
		for f.Next() {
			bases := base.Slice(&f.Cursor)
			for j, id := range f.Cursor.IDs {
				box := space.WrapAABB(geom.NewAABBAt(at[i], 10, 10))
				bases[j].Pos = Position{AABB: box}
				items = append(items, aabbworld.Item{ID: id, Box: box})
				i++
			}
		}
		space.Rebuild(items)
		r.Init(si)
	}})

	r.Draw(nil)
	return counter.n
}

func TestRenderer_Draw_ResolvesOnlyTheEntitiesInView(t *testing.T) {
	quarters := []geom.Vec{geom.NewVec(100, 100), geom.NewVec(700, 100), geom.NewVec(100, 700), geom.NewVec(700, 700)}

	topLeft := camera.NewFromSpace(1000, 1000, 0, geom.NewAABBAt(geom.NewVec(0, 0), 500, 500))
	if n := drawScene(t, 0, topLeft, quarters...); n != 1 {
		t.Errorf("a camera on one quarter resolved %d entities, want the 1 in it", n)
	}

	whole := camera.NewFromSpace(1000, 1000, 0)
	if n := drawScene(t, 0, whole, quarters...); n != 4 {
		t.Errorf("a camera on the whole world resolved %d entities, want all 4", n)
	}
}

func TestRenderer_Draw_SeesAnEntityAcrossTheSeam(t *testing.T) {
	// The box straddles the left edge of a torus: its main piece sits at the right edge.
	onSeam := plane.NewAABB(geom.NewVec(-5, 100), 10, 10)
	rightEdge := camera.NewFromSpace(1000, 1000, aabbworld.Torus, geom.NewAABBAt(geom.NewVec(800, 0), 200, 200))
	if n := drawScene(t, aabbworld.Torus, rightEdge, onSeam.TopLeft); n != 1 {
		t.Errorf("a camera at the right edge resolved %d entities, want the one wrapped onto it", n)
	}
	leftEdge := camera.NewFromSpace(1000, 1000, aabbworld.Torus, geom.NewAABBAt(geom.NewVec(0, 0), 200, 200))
	if n := drawScene(t, aabbworld.Torus, leftEdge, onSeam.TopLeft); n != 1 {
		t.Errorf("a camera at the left edge resolved %d entities, want the one whose piece wrapped in", n)
	}
}

package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/render"
	"github.com/kjkrol/uid"
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

// drawThrough spawns one 10x10 entity per position, lets pick say which of them the View holds
// (nil: the zero View, which sees everything), draws once and returns how many entities had
// their layers resolved.
func drawThrough(t *testing.T, pick func(ids []uid.UID64, v *View), at ...geom.Vec) int {
	t.Helper()
	view := &View{}
	cam := camera.NewFromSpace(1000, 1000, 0)
	r := newRenderer(cam, flatAtlas{}, view, 1000, 1000)
	counter := &countingModifier{}
	r.WithModifier(counter)

	var base goke.Comp[Base]
	var appearance goke.Comp[Appearance]
	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&base, &appearance)
		f.Create(len(at))
		var ids []uid.UID64
		i := 0
		for f.Next() {
			bases := base.Slice(&f.Cursor)
			for j, id := range f.Cursor.IDs {
				bases[j].Pos = Position{AABB: plane.NewAABB(at[i], 10, 10)}
				ids = append(ids, id)
				i++
			}
		}
		if pick != nil {
			pick(ids, view)
		}
		r.Init(si)
	}})

	r.Draw(nil)
	return counter.n
}

func TestRenderer_Draw_ResolvesOnlyWhatTheViewContains(t *testing.T) {
	quarters := []geom.Vec{geom.NewVec(100, 100), geom.NewVec(700, 100), geom.NewVec(100, 700), geom.NewVec(700, 700)}

	firstOnly := func(ids []uid.UID64, v *View) {
		v.Culled = true
		v.In.Add(ids[0])
	}
	if n := drawThrough(t, firstOnly, quarters...); n != 1 {
		t.Errorf("a View holding one entity had %d resolved, want 1", n)
	}
	if n := drawThrough(t, nil, quarters...); n != 4 {
		t.Errorf("the zero View had %d entities resolved, want all 4", n)
	}
}

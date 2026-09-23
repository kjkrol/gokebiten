package board

import (
	"math"
	"time"

	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*terrainBodies)(nil)

// terrainBodies keeps one Body per merged run of impassable cells (immovable) or opaque cells
// (sight only), rebuilt whenever the terrain's version moves — including once after a load, where
// the saved bodies are replaced.
type terrainBodies struct {
	brd         *Board
	worldPlugin *world.Plugin
	typeID      kind.ID

	body   plugin.Tag[Family]
	solid  *world.Bodies
	opaque *world.Bodies
	query  *goke.Query
	base   goke.Comp[world.Base]
	marks  goke.Comp[plugin.Tags[Family]]

	// Each factory gets columns of its own: a Comp handle serves one archetype.
	solidMarks  goke.Comp[plugin.Tags[Family]]
	opaqueMarks goke.Comp[plugin.Tags[Family]]
	collider    goke.Comp[collision.Collider]
	physics     goke.Comp[collision.Physics]

	seen   uint64
	boxes  []bodyBox
	planes []plane.AABB
	ids    []uid.UID64
}

func newTerrainBodies(brd *Board, worldPlugin *world.Plugin, typeID kind.ID, body plugin.Tag[Family]) *terrainBodies {
	return &terrainBodies{brd: brd, worldPlugin: worldPlugin, typeID: typeID, body: body, seen: math.MaxUint64}
}

func (s *terrainBodies) Init(si *goke.SysInit) {
	s.solid = s.worldPlugin.NewBodies(si, s.typeID, &s.collider, &s.physics, &s.solidMarks)
	s.opaque = s.worldPlugin.NewBodies(si, s.typeID, &s.opaqueMarks)
	s.query = si.NewQueryBuilder(&s.base, &s.marks).Build()
	if len(s.present()) == 0 {
		s.spawn()
	}
}

func (s *terrainBodies) Update(cb *goke.CmdBuf, _ time.Duration) {
	if s.brd.Version() == s.seen {
		return
	}
	for _, id := range s.present() {
		s.solid.Remove(cb, id)
	}
	s.spawn()
}

// present lists every body there is right now.
func (s *terrainBodies) present() []uid.UID64 {
	s.ids = s.ids[:0]
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		marks := s.marks.Slice(cursor)
		for i, id := range cursor.IDs {
			if marks[i].Has(s.body) {
				s.ids = append(s.ids, id)
			}
		}
	}
	return s.ids
}

// spawn materializes the terrain as it stands and remembers which version that was.
func (s *terrainBodies) spawn() {
	s.boxes = terrainBoxes(s.brd, s.boxes)
	tagged := plugin.Tags[Family](0).With(s.body)
	s.solid.Spawn(s.planesOf(true), func(i int, _ uid.UID64, cursor *goke.Cursor) {
		s.collider.Slice(cursor)[i] = collision.Collider{}
		s.physics.Slice(cursor)[i] = collision.Physics{Mass: math.Inf(1)}
		s.solidMarks.Slice(cursor)[i] = tagged
	})
	s.opaque.Spawn(s.planesOf(false), func(i int, _ uid.UID64, cursor *goke.Cursor) { s.opaqueMarks.Slice(cursor)[i] = tagged })
	s.seen = s.brd.Version()
}

// planesOf lists the boxes of the solid or the opaque bodies as space rectangles.
func (s *terrainBodies) planesOf(solid bool) []plane.AABB {
	s.planes = s.planes[:0]
	for _, b := range s.boxes {
		if b.solid != solid {
			continue
		}
		size := b.box.BottomRight.Sub(b.box.TopLeft)
		s.planes = append(s.planes, plane.NewAABB(b.box.TopLeft, size.X, size.Y))
	}
	return s.planes
}

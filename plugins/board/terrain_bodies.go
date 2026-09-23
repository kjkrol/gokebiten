package board

import (
	"math"
	"time"

	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
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

	solid    *world.Bodies
	opaque   *world.Bodies
	query    *goke.Query
	base     goke.Comp[world.Base]
	body     goke.Comp[Body]
	collider goke.Comp[collision.Collider]
	physics  goke.Comp[collision.Physics]

	seen   uint64
	boxes  []bodyBox
	planes []plane.AABB
	ids    []uid.UID64
}

func newTerrainBodies(brd *Board, worldPlugin *world.Plugin, typeID kind.ID) *terrainBodies {
	return &terrainBodies{brd: brd, worldPlugin: worldPlugin, typeID: typeID, seen: math.MaxUint64}
}

func (s *terrainBodies) Init(si *goke.SysInit) {
	s.solid = s.worldPlugin.NewBodies(si, s.typeID, &s.collider, &s.physics, &s.body)
	s.opaque = s.worldPlugin.NewBodies(si, s.typeID, &s.body)
	s.query = si.NewQueryBuilder(&s.base).Include(goke.Include[Body]()).Build()
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

// present lists every Body there is right now.
func (s *terrainBodies) present() []uid.UID64 {
	s.ids = s.ids[:0]
	s.query.All()
	for s.query.Next() {
		s.ids = append(s.ids, s.query.Cursor().IDs...)
	}
	return s.ids
}

// spawn materializes the terrain as it stands and remembers which version that was.
func (s *terrainBodies) spawn() {
	s.boxes = terrainBoxes(s.brd, s.boxes)
	s.solid.Spawn(s.planesOf(true), func(i int, _ uid.UID64, cursor *goke.Cursor) {
		s.collider.Slice(cursor)[i] = collision.Collider{}
		s.physics.Slice(cursor)[i] = collision.Physics{Mass: math.Inf(1)}
	})
	s.opaque.Spawn(s.planesOf(false), func(int, uid.UID64, *goke.Cursor) {})
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

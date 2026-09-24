package board

import (
	"math"
	"time"

	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/vision"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*terrainBodySystem)(nil)

// terrainBodySystem keeps one Body per merged run of impassable cells (immovable) or veiled cells
// (sight only), rebuilt whenever the terrain's version moves — including once after a load, where
// the saved bodies are replaced.
type terrainBodySystem struct {
	brd         *Board
	worldPlugin *world.Plugin
	typeID      kind.ID

	body   plugin.Tag[Family]
	solid  *world.Bodies
	veiled *world.Bodies
	query  *goke.Query
	base   goke.Comp[world.Base]
	marks  goke.Comp[plugin.Tags[Family]]

	// Each factory gets columns of its own: a Comp handle serves one archetype.
	solidMarks   goke.Comp[plugin.Tags[Family]]
	veiledMarks  goke.Comp[plugin.Tags[Family]]
	collider     goke.Comp[collision.Collider]
	physics      goke.Comp[collision.Physics]
	transparency goke.Comp[vision.Transparency]

	seen   uint64
	boxes  []bodyBox
	planes []plane.AABB
	veils  []float64
	allows []Domain
	ids    []uid.UID64
}

func newTerrainBodySystem(brd *Board, worldPlugin *world.Plugin, typeID kind.ID, body plugin.Tag[Family]) *terrainBodySystem {
	return &terrainBodySystem{brd: brd, worldPlugin: worldPlugin, typeID: typeID, body: body, seen: math.MaxUint64}
}

func (s *terrainBodySystem) Init(si *goke.SysInit) {
	s.solid = s.worldPlugin.NewBodies(si, s.typeID, &s.collider, &s.physics, &s.solidMarks)
	s.veiled = s.worldPlugin.NewBodies(si, s.typeID, &s.veiledMarks, &s.transparency)
	s.query = si.NewQueryBuilder(&s.base, &s.marks).Build()
	if len(s.present()) == 0 {
		s.spawn()
	}
}

func (s *terrainBodySystem) Update(cb *goke.CmdBuf, _ time.Duration) {
	if s.brd.Version() == s.seen {
		return
	}
	for _, id := range s.present() {
		s.solid.Remove(cb, id)
	}
	s.spawn()
}

// present lists every body there is right now.
func (s *terrainBodySystem) present() []uid.UID64 {
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
func (s *terrainBodySystem) spawn() {
	s.boxes = terrainBoxes(s.brd, s.boxes)
	tagged := plugin.Tags[Family](0).With(s.body)
	n := 0
	s.solid.Spawn(s.planesOf(true), func(i int, _ uid.UID64, cursor *goke.Cursor) {
		// a body pushes whoever its kind does not admit: a wall admitting Air lets a flyer over
		s.collider.Slice(cursor)[i] = collision.Collider{Layers: ^uint8(s.allows[n])}
		s.physics.Slice(cursor)[i] = collision.Physics{Mass: math.Inf(1)}
		s.solidMarks.Slice(cursor)[i] = tagged
		n++
	})
	n = 0
	s.veiled.Spawn(s.planesOf(false), func(i int, _ uid.UID64, cursor *goke.Cursor) {
		s.veiledMarks.Slice(cursor)[i] = tagged
		s.transparency.Slice(cursor)[i] = vision.Transparency{Value: 1 - min(s.veils[n], 1)}
		n++
	})
	s.seen = s.brd.Version()
}

// planesOf lists the boxes of the solid or the veiled bodies as space rectangles, with their veils
// and the domains their kind admits, in the same order.
func (s *terrainBodySystem) planesOf(solid bool) []plane.AABB {
	s.planes, s.veils, s.allows = s.planes[:0], s.veils[:0], s.allows[:0]
	for _, b := range s.boxes {
		if b.solid != solid {
			continue
		}
		size := b.box.BottomRight.Sub(b.box.TopLeft)
		s.planes = append(s.planes, plane.NewAABB(b.box.TopLeft, size.X, size.Y))
		s.veils = append(s.veils, b.veil)
		s.allows = append(s.allows, b.allows)
	}
	return s.planes
}

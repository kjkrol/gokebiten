package board

import (
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

// Ground is the terrain of a cell held as a component on the cell's entity — see
// Plugin.CellEntity; while the entity exists its Ground is what the TerrainMap shows.
type Ground struct{ Kind CellKind }

var _ goke.System = (*cellEntitySystem)(nil)

// cellEntitySystem spawns an entity per cell on request and copies each one's Ground into the
// terrain every tick, so a change to the component is a change to the board.
type cellEntitySystem struct {
	brd         *Board
	worldPlugin *world.Plugin
	typeID      kind.ID

	bodies *world.Bodies
	query  *goke.Query
	cell   goke.Comp[Cell]
	ground goke.Comp[Ground]

	// The factory's own columns: a Comp handle serves one archetype or query, never two.
	spawnCell   goke.Comp[Cell]
	spawnGround goke.Comp[Ground]
}

func newCellEntitySystem(brd *Board, worldPlugin *world.Plugin, typeID kind.ID) *cellEntitySystem {
	return &cellEntitySystem{brd: brd, worldPlugin: worldPlugin, typeID: typeID}
}

func (s *cellEntitySystem) Init(si *goke.SysInit) {
	s.bodies = s.worldPlugin.NewBodies(si, s.typeID, &s.spawnCell, &s.spawnGround)
	s.query = si.NewQueryBuilder(&s.cell, &s.ground).Build()
}

func (s *cellEntitySystem) Update(*goke.CmdBuf, time.Duration) {
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		cells := s.cell.Slice(cursor)
		grounds := s.ground.Slice(cursor)
		for i := range cursor.IDs {
			s.brd.Set(cells[i].ID, grounds[i].Kind)
		}
	}
}

// entity finds the cell's entity or spawns one over the cell, carrying its terrain as Ground.
func (s *cellEntitySystem) entity(c CellID) uid.UID64 {
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		for i, id := range cursor.IDs {
			if s.cell.Slice(cursor)[i].ID == c {
				return id
			}
		}
	}
	w, h := s.brd.CellBounds()
	center := s.brd.CellCenter(c)
	box := plane.NewAABB(geom.NewVec(center.X-w/2, center.Y-h/2), w, h)
	var id uid.UID64
	s.bodies.Spawn([]plane.AABB{box}, func(i int, spawned uid.UID64, cursor *goke.Cursor) {
		s.spawnCell.Slice(cursor)[i] = Cell{ID: c}
		s.spawnGround.Slice(cursor)[i] = Ground{Kind: s.brd.Kind(c)}
		id = spawned
	})
	return id
}

// drop writes a cell entity's Ground into the terrain one last time and despawns it — the
// entity may be gone before the next copy, so the terrain keeps what its Ground last said.
// Any other entity is left alone.
func (s *cellEntitySystem) drop(cb *goke.CmdBuf, id uid.UID64) {
	if !s.query.Seek(id) {
		return
	}
	cursor := s.query.Cursor()
	s.brd.Set(s.cell.At(cursor).ID, s.ground.At(cursor).Kind)
	s.bodies.Remove(cb, id)
}

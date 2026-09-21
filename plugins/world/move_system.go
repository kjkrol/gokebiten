package world

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/uid"
	"math"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
)

var _ goke.System = (*MoveSystem)(nil)

// MoveSystem integrates each entity's already speed-scaled Velocity into
// Position, translating through space (which keeps its spatial index in sync).
type MoveSystem struct {
	space     *aabbworld.Space
	moveQuery *goke.Query
	base      goke.Comp[Base]
	exits     *exits
	leave     func(t plugin.Tick, id uid.UID64)
}

// NewMoveSystem builds world's movement system; no entity moves past its Position.MaxStep a tick.
func NewMoveSystem(space *aabbworld.Space) *MoveSystem {
	return &MoveSystem{space: space, exits: &exits{}, leave: func(plugin.Tick, uid.UID64) {}}
}

func (s *MoveSystem) Init(si *goke.SysInit) {
	s.moveQuery = si.NewQueryBuilder(&s.base).Build()
}

func (s *MoveSystem) Update(cb *goke.CmdBuf, d time.Duration) {
	dt := d.Seconds()
	moved := false
	s.moveQuery.All()
	for s.moveQuery.Next() {
		cursor := s.moveQuery.Cursor()
		bases := s.base.Slice(cursor)
		for i, id := range cursor.IDs {
			rate := bases[i].Vel.Delta()
			step := clampStep(geom.NewVec(rate.X*dt, rate.Y*dt), bases[i].Pos.MaxStep())
			if step.X == 0 && step.Y == 0 {
				continue
			}

			inside := s.space.Translate(id, &bases[i].Pos.AABB, step)
			if (!inside || !s.exits.quiet()) && s.exits.left(id, inside) {
				s.leave(plugin.Tick{Cmd: cb, Now: time.Now(), Dt: d}, id)
			}
			moved = true
		}
	}
	if moved {
		s.space.Flush(nil)
	}
}

// clampStep scales step down to magnitude max if it exceeds it.
func clampStep(step geom.Vec, max float64) geom.Vec {
	mag := math.Hypot(step.X, step.Y)
	if mag <= max || mag == 0 {
		return step
	}
	scale := max / mag
	return geom.NewVec(step.X*scale, step.Y*scale)
}

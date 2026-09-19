package world

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
)

var _ goke.System = (*MoveSystem)(nil)

// MoveSystem integrates each entity's already speed-scaled Velocity into
// Position, translating through space (which keeps its spatial index in sync).
type MoveSystem struct {
	space     *gokg.Space
	maxDelta  float64
	moveQuery *goke.Query
	pos       goke.Comp[Position]
	vel       goke.Comp[Velocity]
}

// NewMoveSystem builds world's movement system, capping per-tick displacement
// to maxDelta (0 for no limit) so nothing can tunnel through another entity.
func NewMoveSystem(space *gokg.Space, maxDelta float64) *MoveSystem {
	return &MoveSystem{space: space, maxDelta: maxDelta}
}

func (s *MoveSystem) Init(si *goke.SysInit) {
	s.moveQuery = si.NewQueryBuilder(&s.pos, &s.vel).Build()
}

func (s *MoveSystem) Update(_ *goke.CmdBuf, d time.Duration) {
	dt := d.Seconds()
	moved := false
	s.moveQuery.All()
	for s.moveQuery.Next() {
		cursor := s.moveQuery.Cursor()
		pos := s.pos.Slice(cursor)
		vel := s.vel.Slice(cursor)
		for i, id := range cursor.IDs {
			rate := vel[i].Delta()
			step := geom.NewVec(rate.X*dt, rate.Y*dt)

			if s.maxDelta > 0 {
				step = clampStep(step, s.maxDelta)
			}
			if step.X == 0 && step.Y == 0 {
				continue
			}

			s.space.Translate(id, &pos[i].AABB, step)
			moved = true
		}
	}
	if moved {
		s.space.Flush(nil)
	}
}

// clampStep scales a step down to magnitude max if it exceeds it, so nothing
// can cross another entity in one tick without ever overlapping it.
func clampStep(step geom.Vec, max float64) geom.Vec {
	mag := math.Hypot(step.X, step.Y)
	if mag <= max || mag == 0 {
		return step
	}
	scale := max / mag
	return geom.NewVec(step.X*scale, step.Y*scale)
}

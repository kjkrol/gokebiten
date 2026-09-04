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
	maxDelta  uint32
	moveQuery *goke.Query
	pos       goke.Comp[Position]
	vel       goke.Comp[Velocity]
}

// NewMoveSystem builds world's movement system, capping per-tick displacement
// to maxDelta (0 for no limit) so nothing can tunnel through another entity.
func NewMoveSystem(space *gokg.Space, maxDelta uint32) *MoveSystem {
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
			vel[i].AccX += float64(rate.X) * dt
			vel[i].AccY += float64(rate.Y) * dt

			if s.maxDelta > 0 {
				clampAcc(&vel[i].AccX, &vel[i].AccY, s.maxDelta)
			}

			dx := int32(vel[i].AccX)
			dy := int32(vel[i].AccY)

			if dx != 0 {
				vel[i].AccX -= float64(dx)
			}
			if dy != 0 {
				vel[i].AccY -= float64(dy)
			}

			if dx != 0 || dy != 0 {
				delta := geom.NewVec(uint32(dx), uint32(dy))
				s.space.Translate(id, &pos[i].AABB, delta)
				moved = true
			}
		}
	}
	if moved {
		s.space.Flush(nil)
	}
}

// clampAcc scales (accX,accY) down to magnitude max if it exceeds it.
func clampAcc(accX, accY *float64, max uint32) {
	mag := math.Hypot(*accX, *accY)
	if mag <= float64(max) {
		return
	}
	scale := float64(max) / mag
	*accX *= scale
	*accY *= scale
}

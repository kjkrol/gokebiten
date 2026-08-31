package world

import (
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
	moveQuery *goke.Query
	pos       goke.Comp[Position]
	vel       goke.Comp[Velocity]
}

// NewMoveSystem builds world's movement system.
func NewMoveSystem(space *gokg.Space) *MoveSystem {
	return &MoveSystem{space: space}
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

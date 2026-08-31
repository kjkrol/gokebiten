package world

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.System = (*VelocitySystem)(nil)

// VelocitySystem folds every registered SpeedModifier's factor into each
// entity's Velocity.Value, then clamps it to maxSpeed (0 for no limit).
type VelocitySystem struct {
	modifiers []SpeedModifier
	maxSpeed  int32
	query     *goke.Query
	vel       goke.Comp[Velocity]
}

func NewVelocitySystem(modifiers []SpeedModifier, maxSpeed int32) *VelocitySystem {
	return &VelocitySystem{modifiers: modifiers, maxSpeed: maxSpeed}
}

func (s *VelocitySystem) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&s.vel)
	for _, m := range s.modifiers {
		m.Bind(qb)
	}
	s.query = qb.Build()
}

func (s *VelocitySystem) Update(_ *goke.CmdBuf, _ time.Duration) {
	if len(s.modifiers) == 0 && s.maxSpeed <= 0 {
		return
	}
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		vel := s.vel.Slice(cursor)
		for i := range cursor.IDs {
			acc := 1.0
			for _, m := range s.modifiers {
				acc = m.Apply(cursor, i, acc)
			}
			vel[i].Value = clampSpeed(int32(float64(vel[i].Value)*acc), s.maxSpeed)
		}
	}
}

// clampSpeed caps a non-negative magnitude to max (0 or less means no limit).
func clampSpeed(v, max int32) int32 {
	if max > 0 && v > max {
		return max
	}
	return v
}

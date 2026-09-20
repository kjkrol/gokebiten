package world

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.System = (*VelocitySystem)(nil)

// VelocitySystem folds every registered SpeedModifier's factor into each entity's Velocity.Value.
type VelocitySystem struct {
	modifiers []SpeedModifier
	query     *goke.Query
	base      goke.Comp[Base]
}

func NewVelocitySystem(modifiers []SpeedModifier) *VelocitySystem {
	return &VelocitySystem{modifiers: modifiers}
}

func (s *VelocitySystem) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&s.base)
	for _, m := range s.modifiers {
		m.Bind(qb)
	}
	s.query = qb.Build()
}

func (s *VelocitySystem) Update(_ *goke.CmdBuf, _ time.Duration) {
	if len(s.modifiers) == 0 {
		return
	}
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		bases := s.base.Slice(cursor)
		for i := range cursor.IDs {
			acc := 1.0
			for _, m := range s.modifiers {
				acc = m.Apply(cursor, i, &bases[i], acc)
			}
			bases[i].Vel.Value = bases[i].Vel.Value * acc
		}
	}
}

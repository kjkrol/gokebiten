package world

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg/geom"
)

var _ goke.System = (*SteeringSystem)(nil)

// SteeringSystem carries out standing Steering requests: it swings Velocity.Dir
// towards Want by no more than TurnRate, and counts down each entity's Reflex
// until its Pending request takes Want's place. It runs between the decision
// pass and movement, so a request made this tick is acted on this tick once
// its Reflex has run out.
type SteeringSystem struct {
	query *goke.Query
	steer goke.Comp[Steering]
	base  goke.Comp[Base]
}

func NewSteeringSystem() *SteeringSystem { return &SteeringSystem{} }

func (s *SteeringSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.steer, &s.base).Build()
}

func (s *SteeringSystem) Update(*goke.CmdBuf, time.Duration) {
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		steers := s.steer.Slice(cursor)
		bases := s.base.Slice(cursor)

		for i := range cursor.IDs {
			st := &steers[i]
			if st.Want.X != 0 || st.Want.Y != 0 {
				bases[i].Vel.Dir = turnTowards(bases[i].Vel.Dir, st.Want, st.TurnRate)
			}
			if st.Delay > 0 {
				st.Delay--
				if st.Delay == 0 {
					st.Want = st.Pending
				}
			}
		}
	}
}

// turnTowards rotates from towards to by at most rate radians, the shorter way
// round. A zero rate, or an entity not heading anywhere yet, snaps straight to
// the target.
func turnTowards(from, to geom.Vec, rate float64) geom.Vec {
	if rate <= 0 || (from.X == 0 && from.Y == 0) {
		return to
	}
	at := math.Atan2(from.Y, from.X)
	delta := math.Mod(math.Atan2(to.Y, to.X)-at+3*math.Pi, 2*math.Pi) - math.Pi
	if math.Abs(delta) <= rate {
		return to
	}
	at += math.Copysign(rate, delta)
	return geom.NewVec(math.Cos(at), math.Sin(at))
}

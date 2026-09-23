package world

import (
	"math"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
)

var _ goke.System = (*SteeringSystem)(nil)

// SteeringSystem carries out Steering requests between the decision pass and movement: heading by
// at most TurnRate a tick, and base speed rewritten each tick for an entity with a motion profile.
type SteeringSystem struct {
	query *goke.Query
	steer goke.Comp[Steering]
	base  goke.Comp[Base]
}

func NewSteeringSystem() *SteeringSystem { return &SteeringSystem{} }

func (s *SteeringSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.steer, &s.base).Build()
}

func (s *SteeringSystem) Update(_ *goke.CmdBuf, d time.Duration) {
	dt := d.Seconds()
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
			if st.MaxSpeed > 0 {
				st.advance(dt)
				bases[i].Vel.Value = st.Speed
			}
		}
	}
}

// turnTowards rotates from towards to by at most rate radians; a zero rate or heading snaps to it.
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

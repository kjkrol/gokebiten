package vision

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/raycast"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*ScanSystem)(nil)

// ScanSystem fills every Sight-carrying entity's Sighted, and the outline of
// those that also carry SightOutline.
//
// Both come off one scan: gathering the candidates is what costs, so splitting
// the outline into a second system would scan the same entity twice a tick.
type ScanSystem struct {
	space *gokg.Space
	view  raycast.View // one for the whole system — see Update

	query   *goke.Query
	sight   goke.Comp[Sight]
	sighted goke.Comp[Sighted]
	outline goke.OptComp[SightOutline]
}

func NewScanSystem(space *gokg.Space) *ScanSystem { return &ScanSystem{space: space} }

func (s *ScanSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.sight, &s.sighted).Optional(&s.outline).Build()
}

func (s *ScanSystem) Update(*goke.CmdBuf, time.Duration) {
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		sights := s.sight.Slice(cursor)
		seen := s.sighted.Slice(cursor)

		// Present asks about the archetype, not the entity, so the branch
		// lifts out of the inner loop.
		var outlines []SightOutline
		if s.outline.Present(cursor) {
			outlines = s.outline.Slice(cursor)
		}

		for i, id := range cursor.IDs {
			sight := &sights[i]
			// One View serves every observer in turn: Entities hands over
			// values and Depths copies into the caller's buffer, so nothing a
			// reader keeps points back into it.
			if !s.space.Scan(id, cone(sight), &s.view) {
				seen[i].Count = 0
				continue
			}
			record(&seen[i], &s.view)
			if outlines != nil {
				trace(&outlines[i], &s.view, sight)
			}
		}
	}
}

func cone(s *Sight) raycast.Cone {
	return raycast.Cone{Direction: s.Facing, HalfAngle: s.HalfAngle, Radius: s.Radius}
}

// record keeps the nearest MaxSeen entities; Visible reports nearest first, so
// anything dropped is further away than everything kept.
func record(dst *Sighted, view *raycast.View) {
	dst.Count = 0
	view.Entities(func(id uid.UID64, dist float64) {
		if dst.Count == MaxSeen {
			return
		}
		dst.IDs[dst.Count] = id
		dst.Dists[dst.Count] = float32(dist)
		dst.Count++
	})
}

// trace samples the cone at the resolution its own reach and width call for,
// never more than the buffer holds.
func trace(dst *SightOutline, view *raycast.View, s *Sight) {
	k := samplesFor(s)
	dst.Count = uint8(len(view.Depths(k, dst.Depths[:0])))
}

// samplesFor is the accuracy rule from sight.go solved for a single cone:
// enough samples that the reach drifts by no more than EdgeTolerance at full
// range, capped by the buffer.
func samplesFor(s *Sight) int {
	k := int(math.Ceil(2*s.HalfAngle*s.Radius/EdgeTolerance)) + 1
	return min(max(k, 2), MaxSamples)
}

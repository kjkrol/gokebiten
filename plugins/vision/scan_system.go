package vision

import (
	"math"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*ScanSystem)(nil)

// ScanSystem fills in what every Sight-carrying entity sees,
// and the outline of those that also carry SightOutline.
type ScanSystem struct {
	space *aabbworld.Space
	view  aabbworld.View // one for the whole system — see Update

	query   *goke.Query
	sight   goke.Comp[Sight]
	base    goke.Comp[world.Base]
	steer   goke.OptComp[world.Steering]
	outline goke.OptComp[SightOutline]

	// lookup resolves a sighted id back to the entity and what it carries.
	lookup     *goke.Query
	lookupBase goke.Comp[world.Base]
	lookupHot  bool

	// host runs the Between behaviors registered with the plugin, inside this pass.
	host *plugin.PairHost[Sighting]

	// What the host is being run over: the observer in hand, everyone it sees, and their tags.
	observer   Sighting
	seen       []Seen
	seenTags   []uint64
	matched    []Seen
	sightingOf func(matched []int) Sighting
}

// The two queries the scan offers its hosted behaviors, by index.
const (
	walked = iota // the observer, a chunk at a time
	sought        // what it sees, one entity at a time
)

func NewScanSystem(space *aabbworld.Space) *ScanSystem {
	return newScanSystem(space, &plugin.PairHost[Sighting]{})
}

func newScanSystem(space *aabbworld.Space, host *plugin.PairHost[Sighting]) *ScanSystem {
	s := &ScanSystem{space: space, host: host}
	s.sightingOf = s.sighting
	return s
}

func (s *ScanSystem) Init(si *goke.SysInit) {
	walk := si.NewQueryBuilder(&s.sight, &s.base).Optional(&s.outline, &s.steer)
	seek := si.NewQueryBuilder(&s.lookupBase)
	s.host.Bind(walk, seek)
	s.query, s.lookup = walk.Build(), seek.Build()
}

func (s *ScanSystem) Update(cb *goke.CmdBuf, d time.Duration) {
	t := plugin.Tick{Cmd: cb, Now: time.Now(), Dt: d}
	hosting := !s.host.Empty()
	s.lookupHot = false

	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		sights := s.sight.Slice(cursor)
		bases := s.base.Slice(cursor)
		steers := s.steer.Slice(cursor)
		var observerTags uint64
		if hosting {
			observerTags = s.host.InChunk(walked, cursor)
		}

		var outlines []SightOutline
		if s.outline.Present(cursor) {
			outlines = s.outline.Slice(cursor)
		}

		for i, id := range cursor.IDs {
			sight := &sights[i]
			if s.space.Scan(id, cone(sight), &s.view) {
				record(&sight.Seen, &s.view)
				if outlines != nil {
					trace(&outlines[i], &s.view, sight)
				}
			} else {
				sight.Seen.Count = 0
			}
			if hosting {
				s.observer = Sighting{Self: id, Base: &bases[i], Sight: sight}
				if i < len(steers) {
					s.observer.Steering = &steers[i]
				}
				s.gather(&sight.Seen)
				s.host.DispatchGrouped(t, observerTags, s.seenTags, s.sightingOf)
			}
		}
	}
}

// gather looks up everyone in found, keeping who is still there and the tags each carries.
func (s *ScanSystem) gather(found *Sighted) {
	s.seen, s.seenTags = s.seen[:0], s.seenTags[:0]
	for k := range int(found.Count) {
		id := found.IDs[k]
		ok := s.lookupHot && s.lookup.SeekH(id)
		if !ok {
			ok = s.lookup.Seek(id)
			s.lookupHot = ok
		}
		if !ok {
			continue
		}
		cursor := s.lookup.Cursor()
		tags := s.host.At(sought, cursor)
		s.seen = append(s.seen, Seen{ID: id, Base: s.lookupBase.At(cursor), Dist: found.Dists[k], TagSet: s.host.TagSet(tags)})
		s.seenTags = append(s.seenTags, tags)
	}
}

// sighting is the observer in hand, seeing just the entities a behavior asked for.
func (s *ScanSystem) sighting(matched []int) Sighting {
	s.matched = s.matched[:0]
	for _, k := range matched {
		s.matched = append(s.matched, s.seen[k])
	}
	out := s.observer
	out.Seen = s.matched
	return out
}

func cone(s *Sight) aabbworld.Cone {
	return aabbworld.Cone{Direction: s.Facing, HalfAngle: s.HalfAngle, Radius: s.Radius}
}

// record keeps the nearest MaxSeen entities of view.
func record(dst *Sighted, view *aabbworld.View) {
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

// trace samples the cone at the resolution its reach and width call for, within the buffer.
func trace(dst *SightOutline, view *aabbworld.View, s *Sight) {
	k := samplesFor(s)
	dst.Count = uint8(len(view.Depths(k, dst.Depths[:0])))
}

// samplesFor is how many samples keep the reach within EdgeTolerance at full range.
func samplesFor(s *Sight) int {
	k := int(math.Ceil(2*s.HalfAngle*s.Radius/EdgeTolerance)) + 1
	return min(max(k, 2), MaxSamples)
}

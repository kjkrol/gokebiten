package selection

import (
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*SelectionSystem)(nil)

// SelectionSystem turns what an event handler wrote into Resources into the Selected tag on
// Selectable entities — a bit flipped in place, seen the same tick.
type SelectionSystem struct {
	space  *aabbworld.Space
	camera camera.Camera
	state  *Resources
	tags   Tags

	query *goke.Query
	marks goke.Comp[plugin.Tags[Family]]
}

// NewSelectionSystem builds a SelectionSystem driven by state over space and cam.
func NewSelectionSystem(state *Resources, space *aabbworld.Space, cam camera.Camera, tags Tags) *SelectionSystem {
	return &SelectionSystem{state: state, space: space, camera: cam, tags: tags}
}

func (s *SelectionSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.marks).Build()
}

func (s *SelectionSystem) Update(_ *goke.CmdBuf, _ time.Duration) {
	if s.state.PendingIDs != nil {
		ids := s.state.PendingIDs
		s.state.PendingIDs = nil
		hit := make(map[uid.UID64]struct{}, len(ids))
		for _, id := range ids {
			hit[id] = struct{}{}
		}
		s.applySelection(hit, false)
	}
	if s.state.Pending != nil {
		p := s.state.Pending
		s.state.Pending = nil
		box := s.worldBox(p.Start, p.End)
		hit := make(map[uid.UID64]struct{})
		collect := func(id uid.UID64) { hit[id] = struct{}{} }
		s.space.Query(box, aabbworld.AnyCapability, collect)
		s.applySelection(hit, p.Additive)
	}
}

// applySelection tags every hit Selectable entity Selected and, unless additive, untags the rest.
func (s *SelectionSystem) applySelection(hit map[uid.UID64]struct{}, additive bool) {
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		marks := s.marks.Slice(cursor)
		for i, id := range cursor.IDs {
			if !marks[i].Has(s.tags.Selectable) {
				continue
			}
			if _, ok := hit[id]; ok {
				marks[i] = marks[i].With(s.tags.Selected)
			} else if !additive {
				marks[i] = marks[i].Without(s.tags.Selected)
			}
		}
	}
}

func (s *SelectionSystem) worldBox(start, end geom.Vec) geom.AABB {
	x0, y0, x1, y1 := camera.FromScreenRect(s.camera, float32(start.X), float32(start.Y), float32(end.X), float32(end.Y))
	minX, maxX := min(x0, x1), max(x0, x1)
	minY, maxY := min(y0, y1), max(y0, y1)
	width, height := maxX-minX, maxY-minY
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return geom.NewAABBAt(geom.NewVec(float64(minX), float64(minY)), float64(width), float64(height))
}

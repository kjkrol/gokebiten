package selection

import (
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*SelectionSystem)(nil)

// SelectionSystem turns what an event handler wrote into Resources into Selected tags.
type SelectionSystem struct {
	space  *aabbworld.Space
	camera camera.Camera
	state  *Resources

	query        *goke.Query
	present      goke.OptComp[Selected]
	selectedAdd  goke.Comp[Selected]
	addEditor    *goke.Editor
	removeEditor *goke.Editor
}

// NewSelectionSystem builds a SelectionSystem driven by state over space and cam.
func NewSelectionSystem(state *Resources, space *aabbworld.Space, cam camera.Camera) *SelectionSystem {
	return &SelectionSystem{state: state, space: space, camera: cam}
}

func (s *SelectionSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder().Optional(&s.present).Build()
	s.addEditor = s.query.NewEditorBuilder(&s.selectedAdd).Build()
	s.removeEditor = s.query.NewEditorBuilder().Remove(goke.Remove[Selected]()).Build()
}

func (s *SelectionSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
	if s.state.PendingIDs != nil {
		ids := s.state.PendingIDs
		s.state.PendingIDs = nil
		hit := make(map[uid.UID64]struct{}, len(ids))
		for _, id := range ids {
			hit[id] = struct{}{}
		}
		s.applySelection(cb, hit, false)
	}
	if s.state.Pending != nil {
		p := s.state.Pending
		s.state.Pending = nil
		box := s.worldBox(p.Start, p.End)
		hit := make(map[uid.UID64]struct{})
		collect := func(id uid.UID64) { hit[id] = struct{}{} }
		s.space.Query(box, aabbworld.AnyCapability, collect)
		s.applySelection(cb, hit, p.Additive)
	}
}

// applySelection tags every hit entity Selected and, unless additive, untags the rest.
func (s *SelectionSystem) applySelection(cb *goke.CmdBuf, hit map[uid.UID64]struct{}, additive bool) {
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		if !s.present.Present(cursor) {
			var toAdd []uid.UID64
			for _, id := range cursor.IDs {
				if _, ok := hit[id]; ok {
					toAdd = append(toAdd, id)
				}
			}
			if len(toAdd) > 0 {
				buf := s.query.BeginMigrate(cb)
				for _, id := range toAdd {
					buf.Add(id)
				}
				buf.Commit(s.addEditor)
			}
			continue
		}

		if additive {
			continue
		}
		var toRemove []uid.UID64
		for _, id := range cursor.IDs {
			if _, ok := hit[id]; !ok {
				toRemove = append(toRemove, id)
			}
		}
		if len(toRemove) > 0 {
			buf := s.query.BeginMigrate(cb)
			for _, id := range toRemove {
				buf.Add(id)
			}
			buf.Commit(s.removeEditor)
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

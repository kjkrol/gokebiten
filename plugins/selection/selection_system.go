package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*SelectionSystem)(nil)

// SelectionSystem turns Resources into Selected tags — the actual query+migrate
// happens in Update, reading whatever HandleEvents implementation wrote
// into Resources this tick.
type SelectionSystem struct {
	space  *gokg.Space
	camera camera.Camera
	state  *Resources

	query        *goke.Query
	present      goke.OptComp[Selected]
	selectedAdd  goke.Comp[Selected]
	addEditor    *goke.Editor
	removeEditor *goke.Editor
}

// NewSelectionSystem builds a SelectionSystem driven by state, querying space and translating drag boxes through cam.
func NewSelectionSystem(state *Resources, space *gokg.Space, cam camera.Camera) *SelectionSystem {
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
		collect := func(id uid.UID64, _ plane.FragPosition) { hit[id] = struct{}{} }
		s.space.Query(box.AABB, collect)
		box.VisitFragments(func(_ plane.FragPosition, fragBox geom.AABB[uint32]) bool {
			s.space.Query(fragBox, collect)
			return true
		})
		s.applySelection(cb, hit, p.Additive)
	}
}

// applySelection adds Selected to every hit entity that lacks it, and
// (unless additive) removes it from every selected entity not in hit.
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

func (s *SelectionSystem) worldBox(start, end geom.Vec[int32]) plane.AABB[uint32] {
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
	raw := plane.NewAABB(geom.NewVec(uint32(int64(minX)), uint32(int64(minY))), uint32(width), uint32(height))
	return s.space.WrapAABB(raw.AABB)
}

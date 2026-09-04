package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*SelectionSystem)(nil)

// SelectionSystem turns State into Selected tags — the actual query+migrate
// happens in Update, reading whatever HandleEvents implementation wrote
// into State this tick.
type SelectionSystem struct {
	space  *gokg.Space
	camera render.Camera
	state  *State

	query        *goke.Query
	present      goke.OptComp[Selected]
	selectedAdd  goke.Comp[Selected]
	addEditor    *goke.Editor
	removeEditor *goke.Editor
}

// NewSelectionSystem builds a SelectionSystem driven by state, querying space and translating drag boxes through camera.
func NewSelectionSystem(state *State, space *gokg.Space, camera render.Camera) *SelectionSystem {
	return &SelectionSystem{state: state, space: space, camera: camera}
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
		s.space.Query(box, func(id uid.UID64, _ plane.FragPosition) { hit[id] = struct{}{} })
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

func (s *SelectionSystem) worldBox(start, end geom.Vec[int32]) geom.AABB[uint32] {
	x0, y0 := s.camera.FromScreen(float32(start.X), float32(start.Y))
	x1, y1 := s.camera.FromScreen(float32(end.X), float32(end.Y))
	minX, maxX := min(x0, x1), max(x0, x1)
	minY, maxY := min(y0, y1), max(y0, y1)
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX <= minX {
		maxX = minX + 1
	}
	if maxY <= minY {
		maxY = minY + 1
	}
	return geom.AABB[uint32]{TopLeft: geom.NewVec(uint32(minX), uint32(minY)), BottomRight: geom.NewVec(uint32(maxX), uint32(maxY))}
}

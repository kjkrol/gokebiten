package selection

import (
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/players"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*SelectionSystem)(nil)

// SelectionSystem carries out Select commands as the Selected tag on Selectable entities — a bit
// flipped in place, seen the same tick.
type SelectionSystem struct {
	selects *players.Inbox[Select]
	space   *aabbworld.Space
	tags    Tags

	query *goke.Query
	marks goke.Comp[plugin.Tags[Family]]
}

// NewSelectionSystem builds a SelectionSystem draining selects over space.
func NewSelectionSystem(selects *players.Inbox[Select], space *aabbworld.Space, tags Tags) *SelectionSystem {
	return &SelectionSystem{selects: selects, space: space, tags: tags}
}

func (s *SelectionSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.marks).Build()
}

func (s *SelectionSystem) Update(_ *goke.CmdBuf, _ time.Duration) {
	s.selects.Drain(func(i players.Issued[Select]) {
		cmd := i.Command
		hit := make(map[uid.UID64]struct{}, len(cmd.IDs))
		if cmd.IDs != nil {
			for _, id := range cmd.IDs {
				hit[id] = struct{}{}
			}
		} else {
			s.space.Query(cmd.Box, aabbworld.AnyCapability, func(id uid.UID64) { hit[id] = struct{}{} })
		}
		s.applySelection(hit, cmd.Additive)
	})
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

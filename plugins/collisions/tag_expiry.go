package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.System = (*TagExpirySystem[struct{}])(nil)

// TagExpirySystem removes tag component T from any entity once its expiry
// (given by expiresAt) has passed — a zero expiresAt never expires.
type TagExpirySystem[T any] struct {
	expiresAt func(*T) time.Time
	query     *goke.Query
	tag       goke.Comp[T]
	remove    *goke.Editor
}

func NewTagExpirySystem[T any](expiresAt func(*T) time.Time) *TagExpirySystem[T] {
	return &TagExpirySystem[T]{expiresAt: expiresAt}
}

func (s *TagExpirySystem[T]) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.tag).Build()
	s.remove = s.query.NewEditorBuilder().Remove(goke.Remove[T]()).Build()
}

func (s *TagExpirySystem[T]) Update(cb *goke.CmdBuf, _ time.Duration) {
	now := time.Now()
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		tags := s.tag.Slice(cursor)
		buf := s.query.BeginMigrate(cb)
		for i, id := range cursor.IDs {
			exp := s.expiresAt(&tags[i])
			if exp.IsZero() || !now.After(exp) {
				continue
			}
			buf.Add(id)
		}
		buf.Commit(s.remove)
	}
}

// Package hit is a ready-made world.Behavior: an entity that struck something
// goes on showing it, long after the single tick its contact lasted.
//
// Register it with world.Plugin.RegisterBehavior, then give Mark and
// collisions.Contacts (which it reads) to whichever kinds should show hits —
// Overlay is the matching draw strategy for world.Renderer.
package hit

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// Mark is how long this entity shows a hit, and whether one is showing now —
// give it to a kind with k.Const(hit.Mark{Duration: 100 * time.Millisecond}).
type Mark struct {
	Duration      time.Duration
	ExpiresAtNano int64
}

// Active reports whether a hit is showing, as of the last tick Behavior ran.
func (m Mark) Active() bool { return m.ExpiresAtNano != 0 }

var _ world.Behavior = (*Behavior)(nil)

// Behavior marks every entity its Contacts say struck something, for as long
// as that entity's own Duration — or the behavior's, for one that sets none.
type Behavior struct {
	fallback time.Duration

	query    *goke.Query
	mark     goke.Comp[Mark]
	contacts goke.Comp[collisions.Contacts]
}

// New builds the behavior, showing hits for d on entities whose Mark leaves Duration unset.
func New(d time.Duration) *Behavior { return &Behavior{fallback: d} }

func (b *Behavior) Init(si *goke.SysInit) {
	b.query = si.NewQueryBuilder(&b.mark, &b.contacts).Build()
}

func (b *Behavior) Update(*goke.CmdBuf, time.Duration) {
	now := time.Now().UnixNano()
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		marks := b.mark.Slice(cursor)
		contacts := b.contacts.Slice(cursor)
		for i := range cursor.IDs {
			// Clearing here, once per tick, is what lets a renderer read Active
			// instead of asking the clock for every entity it draws.
			switch {
			case contacts[i].Count > 0:
				marks[i].ExpiresAtNano = now + int64(b.durationOf(marks[i]))
			case marks[i].Active() && now > marks[i].ExpiresAtNano:
				marks[i].ExpiresAtNano = 0
			}
		}
	}
}

// durationOf is the entity's own hit duration, or the behavior's for one that sets none.
func (b *Behavior) durationOf(m Mark) time.Duration {
	if m.Duration > 0 {
		return m.Duration
	}
	return b.fallback
}

// Overlay draws with on top of whatever an entity looks like, while its Mark is active.
func Overlay(with world.Appearance) world.AppearanceStrategy[Mark] {
	return world.AppearanceStrategyFn[Mark](func(dst []world.Appearance, m Mark) []world.Appearance {
		if !m.Active() {
			return dst
		}
		return append(dst, with)
	})
}

// Package stats is a ready-made world.Behavior: a running count of how many
// contacts happened, for telemetry.
//
// Register it with world.Plugin.RegisterBehavior; only entities carrying
// collisions.Contacts are counted, and each pair counts once however many of
// its two sides record it.
package stats

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
)

var _ world.Behavior = (*Behavior)(nil)

// Behavior adds this tick's contacts to a Stats counter the game owns.
type Behavior struct {
	stats *Stats

	query    *goke.Query
	contacts goke.Comp[collisions.Contacts]
}

// New builds the behavior, counting into stats.
func New(stats *Stats) *Behavior { return &Behavior{stats: stats} }

func (b *Behavior) Init(si *goke.SysInit) {
	b.query = si.NewQueryBuilder(&b.contacts).Build()
}

func (b *Behavior) Update(*goke.CmdBuf, time.Duration) {
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		contacts := b.contacts.Slice(cursor)
		for i, self := range cursor.IDs {
			for _, c := range contacts[i].All() {
				// The lower index owns the pair — the same rule the narrow
				// phase pairs by, so a contact both sides recorded counts once.
				if self.Index() < c.Other.Index() {
					b.stats.Counter++
				}
			}
		}
	}
}

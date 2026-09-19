// Package debug is a ready-made world.Behavior: it logs every contact, for
// when a collision is easier to read than to watch.
//
// Register it with world.Plugin.RegisterBehavior; only entities carrying
// collisions.Contacts are logged, and each pair is logged once however many of
// its two sides recorded it.
package debug

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// Formatter renders one recorded contact as a log line.
type Formatter func(self uid.UID64, c collisions.Contact) string

func defaultFormat(self uid.UID64, c collisions.Contact) string {
	return fmt.Sprintf("collision: %v <-> %v (impact %.2f)", self, c.Other, c.Impact)
}

var _ world.Behavior = (*Behavior)(nil)

// Behavior writes a line per contact. Zero-config prints to stdout; use
// WithWriter/WithFormat for somewhere else, or for more of the contact.
type Behavior struct {
	w      io.Writer
	format Formatter

	query    *goke.Query
	contacts goke.Comp[collisions.Contacts]
}

type Option func(*Behavior)

func WithWriter(w io.Writer) Option { return func(b *Behavior) { b.w = w } }
func WithFormat(f Formatter) Option { return func(b *Behavior) { b.format = f } }

func New(opts ...Option) *Behavior {
	b := &Behavior{w: os.Stdout, format: defaultFormat}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

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
				// The lower index owns the pair, so a contact both sides
				// recorded is reported once.
				if self.Index() < c.Other.Index() {
					fmt.Fprintln(b.w, b.format(self, c))
				}
			}
		}
	}
}

// Package debug is a ready-made reaction to contacts: it logs them, for when a
// collision is easier to read than to watch.
//
// Hand Log to plugin.Between and register that with
// collisions.Plugin.RegisterBehavior — Between[plugin.Anything, plugin.Anything]
// to log every confirmed contact once, or a pair of tags to log just those. It
// rides the narrow phase's own pass, at the cost of no query of its own.
package debug

import (
	"fmt"
	"io"
	"os"

	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
)

// Formatter renders one contact as a log line.
type Formatter func(m collisions.Meeting) string

func defaultFormat(m collisions.Meeting) string {
	return fmt.Sprintf("collision: %v <-> %v (impact %.2f)", m.Self, m.Other, m.Impact)
}

type config struct {
	w      io.Writer
	format Formatter
}

type Option func(*config)

func WithWriter(w io.Writer) Option { return func(c *config) { c.w = w } }
func WithFormat(f Formatter) Option { return func(c *config) { c.format = f } }

// Log writes a line per contact it is handed — to stdout in the default format,
// unless WithWriter or WithFormat say otherwise.
func Log(opts ...Option) func(plugin.Tick, collisions.Meeting) {
	c := &config{w: os.Stdout, format: defaultFormat}
	for _, opt := range opts {
		opt(c)
	}
	return func(_ plugin.Tick, m collisions.Meeting) { fmt.Fprintln(c.w, c.format(m)) }
}

package host

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
)

// eachRunner is an Each behavior with its component type erased.
type eachRunner[P any] interface {
	bind(qb *goke.QueryBuilder)
	run(t plugin.Tick, cursor *goke.Cursor, about func(i int) P)
}

type each[T, P any] struct {
	state goke.OptComp[T]
	react func(plugin.Tick, *T, P)
}

func (e *each[T, P]) bind(qb *goke.QueryBuilder) { qb.Optional(&e.state) }

func (e *each[T, P]) run(t plugin.Tick, cursor *goke.Cursor, about func(i int) P) {
	if !e.state.Present(cursor) {
		return
	}
	states := e.state.Slice(cursor)
	for i := range cursor.IDs {
		e.react(t, &states[i], about(i))
	}
}

type every[P any] struct{ react func(plugin.Tick, P) }

func (e *every[P]) bind(*goke.QueryBuilder) {}

func (e *every[P]) run(t plugin.Tick, cursor *goke.Cursor, about func(i int) P) {
	for i := range cursor.IDs {
		e.react(t, about(i))
	}
}

// EachHost runs the Each and Every behaviors made for payload P inside a host's own walk
// over its entities.
type EachHost[P any] struct {
	runners []eachRunner[P]
	bound   bool
}

// Empty reports whether no behavior was added.
func (h *EachHost[P]) Empty() bool { return len(h.runners) == 0 }

// Add takes an Each or Every behavior for P; plugin.ErrUnhostedBehavior for another, plugin.ErrHostBuilt after Bind.
func (h *EachHost[P]) Add(b plugin.Behavior) error {
	runner, ok := b.(eachRunner[P])
	if !ok {
		return fmt.Errorf("%w: %T", plugin.ErrUnhostedBehavior, b)
	}
	if h.bound {
		return fmt.Errorf("%w: %T", plugin.ErrHostBuilt, b)
	}
	h.runners = append(h.runners, runner)
	return nil
}

// Bind adds every behavior's component to the host's query — call once, before it is built.
func (h *EachHost[P]) Bind(qb *goke.QueryBuilder) {
	h.bound = true
	for _, r := range h.runners {
		r.bind(qb)
	}
}

// Run runs every behavior over the chunk being walked; about(i) describes its i-th entity.
func (h *EachHost[P]) Run(t plugin.Tick, cursor *goke.Cursor, about func(i int) P) {
	for _, r := range h.runners {
		r.run(t, cursor, about)
	}
}

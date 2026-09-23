package plugin

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
)

// Pair is a Between behavior with its tags erased: what a host finds in RegisterBehavior.
type Pair[P any] struct {
	a, b  tagged
	same  bool
	react func(Tick, P)

	// fa and fb are the host's family indices of a and b; -1 for Any.
	fa, fb int
}

// PairHost runs the Between behaviors made for payload P inside a host's own pass. It reads
// every family its behaviors name off the entities it is shown and matches pairs by tag bits.
type PairHost[P any] struct {
	families []tagProbe
	pairs    []*Pair[P]
	bound    bool
	matched  []int // DispatchGrouped's scratch
}

// Add takes a Between behavior for P; ErrUnhostedBehavior for another, ErrHostBuilt after Bind.
func (h *PairHost[P]) Add(b Behavior) error {
	pair, ok := b.(*Pair[P])
	if !ok {
		return fmt.Errorf("%w: %T", ErrUnhostedBehavior, b)
	}
	if h.bound {
		return fmt.Errorf("%w: %T", ErrHostBuilt, b)
	}
	pair.fa, pair.fb = h.familyOf(pair.a), h.familyOf(pair.b)
	h.pairs = append(h.pairs, pair)
	return nil
}

// Empty reports a host with nothing to run.
func (h *PairHost[P]) Empty() bool { return len(h.pairs) == 0 }

// familyOf is the index of t's family among the host's, adding it on first sight; -1 for Any.
func (h *PairHost[P]) familyOf(t tagged) int {
	if t.family == nil {
		return -1
	}
	for i, known := range h.families {
		if known.family() == t.family {
			return i
		}
	}
	if len(h.families) == MaxFamilies {
		panic(fmt.Sprintf("plugin: pair behaviors name more than %d tag families", MaxFamilies))
	}
	h.families = append(h.families, t.make())
	return len(h.families) - 1
}

// Bind adds every family to each of the host's queries — call once, before they are built.
func (h *PairHost[P]) Bind(queries ...*goke.QueryBuilder) {
	h.bound = true
	for _, f := range h.families {
		f.bind(queries)
	}
}

// InChunk is what the i-th entity of the chunk being walked on query carries.
func (h *PairHost[P]) InChunk(query int, cursor *goke.Cursor, i int) Marks {
	m := Marks{families: &h.families}
	for k, f := range h.families {
		m.words[k] = f.inChunk(query, cursor, i)
	}
	return m
}

// At is what the entity just sought on query carries.
func (h *PairHost[P]) At(query int, cursor *goke.Cursor) Marks {
	m := Marks{families: &h.families}
	for k, f := range h.families {
		m.words[k] = f.at(query, cursor)
	}
	return m
}

// fits reports whether an entity carrying m satisfies side t of a pair, family index f.
func fits(m Marks, f int, t tagged) bool {
	return f < 0 || m.words[f]&(1<<t.bit) != 0
}

// Dispatch runs every behavior whose first tag self carries and second tag other carries.
func (h *PairHost[P]) Dispatch(t Tick, self, other Marks, pair P) {
	for _, b := range h.pairs {
		if fits(self, b.fa, b.a) && fits(other, b.fb, b.b) {
			b.react(t, pair)
		}
	}
}

// DispatchGrouped is Dispatch for one Self against many Others, run even when none match.
func (h *PairHost[P]) DispatchGrouped(t Tick, self Marks, others []Marks, build func(matched []int) P) {
	for _, b := range h.pairs {
		if !fits(self, b.fa, b.a) {
			continue
		}
		h.matched = h.matched[:0]
		for i := range others {
			if fits(others[i], b.fb, b.b) {
				h.matched = append(h.matched, i)
			}
		}
		b.react(t, build(h.matched))
	}
}

// DispatchEitherWay is Dispatch for a pair with no direction, run whichever way the tags fit.
func (h *PairHost[P]) DispatchEitherWay(t Tick, a, b Marks, forward, backward P) {
	for _, p := range h.pairs {
		if fits(a, p.fa, p.a) && fits(b, p.fb, p.b) {
			p.react(t, forward)
			if p.same {
				continue
			}
		}
		if fits(b, p.fa, p.a) && fits(a, p.fb, p.b) {
			p.react(t, backward)
		}
	}
}

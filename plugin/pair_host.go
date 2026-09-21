package plugin

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
)

// MaxTags is how many distinct tags one PairHost's behaviors may name between them.
const MaxTags = 64

// Pair is a Between behavior with its tags erased: what a host finds in RegisterBehavior.
type Pair[P any] struct {
	self, other tagProbe // nil for Anything
	asks        []tagProbe
	same        bool
	react       func(Tick, P)

	wantSelf, wantOther uint64
}

// PairHost runs the Between behaviors made for payload P inside a host's own pass.
// Each tag answers on one bit of a mask the host reads and hands back to Dispatch.
type PairHost[P any] struct {
	tags    []tagProbe
	pairs   []*Pair[P]
	bound   bool
	matched []int // DispatchGrouped's scratch
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
	pair.wantSelf, pair.wantOther = h.bitOf(pair.self), h.bitOf(pair.other)
	for _, asked := range pair.asks {
		h.bitOf(asked)
	}
	h.pairs = append(h.pairs, pair)
	return nil
}

// Empty reports a host with nothing to run.
func (h *PairHost[P]) Empty() bool { return len(h.pairs) == 0 }

// TagSet wraps a mask read with InChunk or At, for a payload to hand on.
func (h *PairHost[P]) TagSet(mask uint64) TagSet { return TagSet{mask: mask, tags: &h.tags} }

// bitOf is the mask bit p's tag answers on, shared by every probe of that tag — zero for Anything.
func (h *PairHost[P]) bitOf(p tagProbe) uint64 {
	if p == nil {
		return 0
	}
	for i, known := range h.tags {
		if known.tag() == p.tag() {
			return 1 << i
		}
	}
	if len(h.tags) == MaxTags {
		panic(fmt.Sprintf("plugin: pair behaviors name more than %d distinct tags", MaxTags))
	}
	h.tags = append(h.tags, p)
	return 1 << (len(h.tags) - 1)
}

// Bind adds every tag to each of the host's queries — call once, before they are built.
func (h *PairHost[P]) Bind(queries ...*goke.QueryBuilder) {
	h.bound = true
	for _, t := range h.tags {
		t.bind(queries)
	}
}

// InChunk is which tags every entity of the chunk being walked on query carries.
func (h *PairHost[P]) InChunk(query int, cursor *goke.Cursor) (mask uint64) {
	for i, t := range h.tags {
		if t.inChunk(query, cursor) {
			mask |= 1 << i
		}
	}
	return mask
}

// At is which tags the entity just sought on query carries.
func (h *PairHost[P]) At(query int, cursor *goke.Cursor) (mask uint64) {
	for i, t := range h.tags {
		if t.at(query, cursor) {
			mask |= 1 << i
		}
	}
	return mask
}

// Dispatch runs every behavior whose Self tag is in self and Other tag in other.
func (h *PairHost[P]) Dispatch(t Tick, self, other uint64, pair P) {
	for _, b := range h.pairs {
		if self&b.wantSelf == b.wantSelf && other&b.wantOther == b.wantOther {
			b.react(t, pair)
		}
	}
}

// DispatchGrouped is Dispatch for one Self against many Others, run even when none match.
func (h *PairHost[P]) DispatchGrouped(t Tick, self uint64, others []uint64, build func(matched []int) P) {
	for _, b := range h.pairs {
		if self&b.wantSelf != b.wantSelf {
			continue
		}
		h.matched = h.matched[:0]
		for i, other := range others {
			if other&b.wantOther == b.wantOther {
				h.matched = append(h.matched, i)
			}
		}
		b.react(t, build(h.matched))
	}
}

// DispatchEitherWay is Dispatch for a pair with no direction, run whichever way the tags fit.
func (h *PairHost[P]) DispatchEitherWay(t Tick, a, b uint64, forward, backward P) {
	for _, p := range h.pairs {
		if a&p.wantSelf == p.wantSelf && b&p.wantOther == p.wantOther {
			p.react(t, forward)
			if p.same {
				continue
			}
		}
		if b&p.wantSelf == p.wantSelf && a&p.wantOther == p.wantOther {
			p.react(t, backward)
		}
	}
}

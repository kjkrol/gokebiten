package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/kjkrol/goke/v3"
)

// Behavior is game logic a Plugin runs inside its own pass over its entities,
// rather than as one more system with a query of its own — built with Between
// or Each, and registered with the plugin it concerns.
//
// Which plugin that is shows in the behavior's payload type: each host defines
// its own, and refuses a behavior made for another's. A reusable strategy is a
// plain func of that payload — the tags it runs between are the registration's
// to name, Between[Wolf, Hare](hunt.Chase(d)).
type Behavior any

// ErrUnhostedBehavior is what RegisterBehavior reports for a behavior the Plugin has no pass to run it in.
var ErrUnhostedBehavior = errors.New("plugin: behavior cannot be hosted here")

// ErrHostBuilt is what registering a behavior reports once its host's queries exist — it could no longer join them.
var ErrHostBuilt = errors.New("plugin: behavior registered after its host was built")

// Tick is what a hosted behavior is told about the pass it runs in.
type Tick struct {
	Cmd *goke.CmdBuf  // structural changes land when the pass is over
	Now time.Time     // read once for the whole pass
	Dt  time.Duration // length of this tick
}

// Anything stands for "whatever it is" on one side of Between, or both.
type Anything struct{}

// Between is a behavior for every pair a host comes across in which one entity
// carries A and the other B — a contact, a sighting, whatever the host pairs up.
// P is the host's own description of the pair, and is what says whose it is.
//
// A and B join the host's queries as optional components, so the behavior costs
// no query of its own. Name A and B; P follows from react. Any tag react means
// to ask about through a TagSet has to be declared here with Asking.
func Between[A, B, P any](react func(t Tick, pair P), asking ...Ask) Behavior {
	pair := &Pair[P]{
		self: probeFor[A](), other: probeFor[B](),
		same:  reflect.TypeFor[A]() == reflect.TypeFor[B](),
		react: react,
	}
	for _, ask := range asking {
		pair.asks = append(pair.asks, ask.probe)
	}
	return pair
}

// Ask is a tag a behavior wants to be able to look for on the entities it is
// handed — see Asking.
type Ask struct{ probe tagProbe }

// Asking declares that a behavior will ask TagSet.Carries about T, so the host
// can bind T to its queries: a tag nobody declared is one it never looks up.
func Asking[T any]() Ask { return Ask{probe: probeFor[T]()} }

// TagSet is which of a host's tags one entity carries — what a payload hands a
// behavior so it can tell the entities it was given apart.
type TagSet struct {
	mask uint64
	tags *[]tagProbe
}

// Carries reports whether the entity carries T — which the behavior must have declared with Asking, or named in Between.
func (s TagSet) Carries[T any]() bool {
	want := reflect.TypeFor[T]()
	for i, known := range *s.tags {
		if known.tag() == want {
			return s.mask&(1<<i) != 0
		}
	}
	panic(fmt.Sprintf("plugin: Carries[%v] asked of a host that was never told about it — declare it with plugin.Asking", want))
}

// Each is a behavior run on every entity a host visits that carries T, with
// whatever the host has to say about that entity — P, which is what says whose
// behavior it is.
//
// T joins the host's query as an optional component, so the behavior costs no
// query of its own. Name T; P follows from react.
func Each[T, P any](react func(t Tick, state *T, about P)) Behavior {
	return &each[T, P]{react: react}
}

// tagProbe answers "does this entity carry the tag" on each of a host's
// queries: by chunk where one is being walked, by entity where one was sought.
type tagProbe interface {
	tag() reflect.Type
	bind(queries []*goke.QueryBuilder)
	inChunk(query int, cursor *goke.Cursor) bool
	at(query int, cursor *goke.Cursor) bool
}

type probe[T any] struct {
	comps []goke.OptComp[T] // one per host query; never grown once bound
}

// probeFor is T's probe, or nil for Anything — which every entity satisfies.
func probeFor[T any]() tagProbe {
	if reflect.TypeFor[T]() == reflect.TypeFor[Anything]() {
		return nil
	}
	return &probe[T]{}
}

func (p *probe[T]) tag() reflect.Type { return reflect.TypeFor[T]() }

func (p *probe[T]) bind(queries []*goke.QueryBuilder) {
	p.comps = make([]goke.OptComp[T], len(queries))
	for i, qb := range queries {
		qb.Optional(&p.comps[i])
	}
}

func (p *probe[T]) inChunk(query int, cursor *goke.Cursor) bool {
	return p.comps[query].Present(cursor)
}

func (p *probe[T]) at(query int, cursor *goke.Cursor) bool {
	return p.comps[query].At(cursor) != nil
}

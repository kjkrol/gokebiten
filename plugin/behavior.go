package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/kjkrol/goke/v3"
)

// Behavior is game logic a Plugin runs inside its own pass over its entities, built with
// Between or Each; its payload type says which plugin hosts it, and another refuses it.
type Behavior any

// ErrUnhostedBehavior is what RegisterBehavior reports for a behavior the Plugin cannot run.
var ErrUnhostedBehavior = errors.New("plugin: behavior cannot be hosted here")

// ErrHostBuilt is what registering a behavior reports once its host's queries exist.
var ErrHostBuilt = errors.New("plugin: behavior registered after its host was built")

// Tick is what a hosted behavior is told about the pass it runs in.
type Tick struct {
	CmdBuf *goke.CmdBuf  // structural changes land when the pass is over
	Now    time.Time     // read once for the whole pass
	Dt     time.Duration // length of this tick
}

// Between is a behavior for every pair a host meets where one entity carries a and the other b;
// Any on a side takes whatever is there.
func Between[FA, FB, P any](a Tag[FA], b Tag[FB], react func(t Tick, pair P)) Behavior {
	return &Pair[P]{
		a: tagOf(a), b: tagOf(b),
		same:  reflect.TypeFor[FA]() == reflect.TypeFor[FB]() && uint8(a) == uint8(b),
		react: react,
	}
}

// Each is a behavior run on every entity a host visits that carries T.
func Each[T, P any](react func(t Tick, state *T, about P)) Behavior {
	return &each[T, P]{react: react}
}

// MaxFamilies is how many tag families one host's behaviors may name between them.
const MaxFamilies = 8

// Marks is which tags of a host's families one entity carries — what a host reads from an
// entity and hands back to Dispatch, and what a payload passes on for Carries.
type Marks struct {
	words    [MaxFamilies]uint64
	families *[]tagProbe
}

// Carries reports whether the entity behind m carries t; the host must name t's family in a
// behavior, or it never read it.
func (m Marks) Carries[F any](t Tag[F]) bool {
	if m.families == nil {
		return false
	}
	want := reflect.TypeFor[F]()
	for i, known := range *m.families {
		if known.family() == want {
			return m.words[i]&(1<<t) != 0
		}
	}
	panic(fmt.Sprintf("plugin: Carries asked about family %v, which no behavior of this host names", want))
}

// tagged is one side of a Pair with its family erased: which family, which bit, and how to
// build the family's probe, since a host cannot from the type alone.
type tagged struct {
	family reflect.Type // nil for Any
	bit    uint8
	make   func() tagProbe
}

func tagOf[F any](t Tag[F]) tagged {
	if reflect.TypeFor[F]() == reflect.TypeFor[Anything]() {
		return tagged{}
	}
	return tagged{family: reflect.TypeFor[F](), bit: uint8(t), make: func() tagProbe { return &probe[F]{} }}
}

// tagProbe reads one family's Tags on each of a host's queries: by chunk where one is being
// walked, by entity where one was sought.
type tagProbe interface {
	family() reflect.Type
	bind(queries []*goke.QueryBuilder)
	inChunk(query int, cursor *goke.Cursor, i int) uint64
	at(query int, cursor *goke.Cursor) uint64
}

type probe[F any] struct {
	comps []goke.OptComp[Tags[F]] // one per host query; never grown once bound
}

func (p *probe[F]) family() reflect.Type { return reflect.TypeFor[F]() }

func (p *probe[F]) bind(queries []*goke.QueryBuilder) {
	p.comps = make([]goke.OptComp[Tags[F]], len(queries))
	for i, qb := range queries {
		qb.Optional(&p.comps[i])
	}
}

func (p *probe[F]) inChunk(query int, cursor *goke.Cursor, i int) uint64 {
	if !p.comps[query].Present(cursor) {
		return 0
	}
	return uint64(p.comps[query].Slice(cursor)[i])
}

func (p *probe[F]) at(query int, cursor *goke.Cursor) uint64 {
	if v := p.comps[query].At(cursor); v != nil {
		return uint64(*v)
	}
	return 0
}

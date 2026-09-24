package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/kjkrol/goke/v3"
)

// Behavior is game logic a Plugin runs inside its own pass over its entities, built with the
// hosting plugin's own constructors (vision.Between, board.Each, world.Every, ...); its payload
// type says which plugin hosts it, and another refuses it.
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

// MaxFamilies is how many tag families one host's behaviors may name between them.
const MaxFamilies = 8

// Marks is which tags of a host's families one entity carries — what a host reads from an
// entity and hands back to Dispatch, and what a payload passes on for Carries.
type Marks struct {
	words    [MaxFamilies]uint64
	families *[]reflect.Type
}

// MarksOf is what a host builds from the words it read, one per family in families' order.
func MarksOf(words [MaxFamilies]uint64, families *[]reflect.Type) Marks {
	return Marks{words: words, families: families}
}

// Word is the bits read for the host's family at index i.
func (m Marks) Word(i int) uint64 { return m.words[i] }

// Carries reports whether the entity behind m carries t; the host must name t's family in a
// behavior, or it never read it.
func (m Marks) Carries[F any](t Tag[F]) bool {
	if m.families == nil {
		return false
	}
	want := reflect.TypeFor[F]()
	for i, known := range *m.families {
		if known == want {
			return m.words[i]&(1<<t) != 0
		}
	}
	panic(fmt.Sprintf("plugin: Carries asked about family %v, which no behavior of this host names", want))
}

package world

import (
	"fmt"

	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/uid"
)

// ComponentTemplate is one component of an EntKind — see Const and Load.
type ComponentTemplate interface {
	accepts(data any) bool
	adder() entityExtras
}

// Template yields one component value per spawned entity, optionally
// running an effect right after it's written.
type Template[T any] struct {
	value  func(data any) T
	check  func(data any) bool
	effect func(v T, id uid.UID64)
}

var _ ComponentTemplate = Template[struct{}]{}

// Const is a Template whose value comes straight from the EntKind.
func Const[T any](v T) Template[T] {
	return Template[T]{
		value: func(any) T { return v },
		check: func(any) bool { return true },
	}
}

// Load is a Template whose value is read from each Roster Entry's Data, which must be a P.
func Load[P, T any](load func(data P) T) Template[T] {
	return Template[T]{
		value: func(data any) T { return load(data.(P)) },
		check: func(data any) bool { _, ok := data.(P); return ok },
	}
}

// WithEffect sets a callback run right after this template's value is written for each spawned entity.
func (t Template[T]) WithEffect(effect func(v T, id uid.UID64)) Template[T] {
	t.effect = effect
	return t
}

func (t Template[T]) accepts(data any) bool { return t.value != nil && t.check(data) }

func (t Template[T]) adder() entityExtras {
	return &componentAdder[T]{value: t.value, effect: t.effect}
}

func (t Template[T]) resolve(data any, id uid.UID64) T {
	v := t.value(data)
	if t.effect != nil {
		t.effect(v, id)
	}
	return v
}

// EntKind identifies a named entity archetype: every component an entity of
// this kind spawns with, each fixed (Const) or read from its Roster Entry (Load).
type EntKind struct {
	Name       string
	SpriteID   render.SpriteID
	Position   Template[Position]
	Velocity   Template[Velocity]
	Components []ComponentTemplate
}

func (k EntKind) validate(data any) error {
	if !k.Position.accepts(data) || !k.Velocity.accepts(data) {
		return fmt.Errorf("world: EntKind %q: Position/Velocity unset or reject Data of type %T", k.Name, data)
	}
	for _, c := range k.Components {
		if !c.accepts(data) {
			return fmt.Errorf("world: EntKind %q: a component template rejects Data of type %T", k.Name, data)
		}
	}
	return nil
}

// EntKindDict is a Plugin's registered set of EntKinds, keyed by Name —
// reached via Plugin.EntKindDict, never built directly by the game.
type EntKindDict interface {
	// Create registers kinds, assigning each one's SpriteID by call order.
	Create(kinds ...EntKind)
	// Get resolves name to the EntKind registered under it.
	Get(name string) (EntKind, bool)
	// All returns every registered EntKind.
	All() []EntKind
}

type entKindDict struct {
	entries map[string]EntKind
	next    render.SpriteID
}

func newEntKindDict() *entKindDict { return &entKindDict{entries: make(map[string]EntKind)} }

func (d *entKindDict) Create(kinds ...EntKind) {
	for _, k := range kinds {
		k.SpriteID = d.next
		d.next++
		d.entries[k.Name] = k
	}
}

func (d *entKindDict) Get(name string) (EntKind, bool) {
	k, ok := d.entries[name]
	return k, ok
}

func (d *entKindDict) All() []EntKind {
	all := make([]EntKind, 0, len(d.entries))
	for _, k := range d.entries {
		all = append(all, k)
	}
	return all
}

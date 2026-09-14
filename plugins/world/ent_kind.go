package world

import (
	"fmt"

	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/uid"
)

// ComponentTemplate is one component of an EntKind — see Const and Kind.Load.
type ComponentTemplate interface {
	adder() entityExtras
}

// Template yields one component value per spawned entity, optionally
// running an effect right after it's written.
type Template[T any] struct {
	value  func(data any) T
	effect func(v T, id uid.UID64)
}

var _ ComponentTemplate = Template[struct{}]{}

// Const is a Template whose value comes straight from the EntKind.
func Const[T any](v T) Template[T] {
	return Template[T]{value: func(any) T { return v }}
}

// Kind carries a kind's roster data type P for defining its templates — see EntKindDict.Define.
type Kind[P any] struct{}

// Load is a Template whose value is read from each of this kind's roster entries.
func (Kind[P]) Load[T any](load func(data P) T) Template[T] {
	return Template[T]{value: func(data any) T { return load(data.(P)) }}
}

// WithEffect sets a callback run right after this template's value is written for each spawned entity.
func (t Template[T]) WithEffect(effect func(v T, id uid.UID64)) Template[T] {
	t.effect = effect
	return t
}

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
// this kind spawns with, each fixed (Const) or read from its roster entry (Kind.Load).
type EntKind struct {
	Name       string
	SpriteID   render.SpriteID
	Position   Template[Position]
	Velocity   Template[Velocity]
	Components []ComponentTemplate

	accepts func(data any) bool
}

func (k EntKind) validate(data any) error {
	if k.Position.value == nil || k.Velocity.value == nil {
		return fmt.Errorf("world: EntKind %q: Position/Velocity unset", k.Name)
	}
	if k.accepts == nil || !k.accepts(data) {
		return fmt.Errorf("world: EntKind %q rejects roster data of type %T", k.Name, data)
	}
	return nil
}

// EntKindDict is a Plugin's registered set of EntKinds, keyed by Name —
// reached via Plugin.EntKindDict, never built directly by the game.
type EntKindDict struct {
	entries map[string]EntKind
	next    render.SpriteID
}

func newEntKindDict() *EntKindDict { return &EntKindDict{entries: make(map[string]EntKind)} }

// Define registers the EntKind define builds, whose roster entries carry a P, assigning its SpriteID by call order.
func (d *EntKindDict) Define[P any](name string, define func(k Kind[P]) EntKind) {
	k := define(Kind[P]{})
	k.Name = name
	k.SpriteID = d.next
	k.accepts = func(data any) bool { _, ok := data.(P); return ok }
	d.next++
	d.entries[k.Name] = k
}

// Entry is a roster entry spawning one entity of the kind named name from data — panics if no such kind or data isn't its P.
func (d *EntKindDict) Entry(name string, data any) Entry {
	k, ok := d.entries[name]
	if !ok {
		panic(fmt.Sprintf("world: unknown EntKind %q", name))
	}
	if err := k.validate(data); err != nil {
		panic(err)
	}
	return Entry{kind: name, data: data}
}

// Get resolves name to the EntKind registered under it.
func (d *EntKindDict) Get(name string) (EntKind, bool) {
	k, ok := d.entries[name]
	return k, ok
}

// All returns every registered EntKind.
func (d *EntKindDict) All() []EntKind {
	all := make([]EntKind, 0, len(d.entries))
	for _, k := range d.entries {
		all = append(all, k)
	}
	return all
}

package world

import (
	"fmt"
	"reflect"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world/kind"
	"github.com/kjkrol/gokebiten/render"
)

// Kinds is a Plugin's registered set of entity kinds — reached via Plugin.Kinds
// and handed to kind.Define, never built directly by the game.
type Kinds struct {
	entries map[string]registered
	next    render.SpriteID
	order   []string // live: kind.ID -> name, in Define order
	saved   []string // what a save brought in; empty on a fresh run
}

// registered is a kind as the world spawns it: its Position and Velocity picked
// out of the Spec, since those two live in Base rather than in columns of their own.
type registered struct {
	name     string
	typeID   kind.ID
	spriteID render.SpriteID
	row      reflect.Type
	position kind.Template[Position]
	velocity kind.Template[Velocity]
	comps    []kind.Comp
}

var (
	_ kind.Registry       = (*Kinds)(nil)
	_ goke.SetupProvider  = (*Kinds)(nil)
	_ goke.CompProvider   = (*Kinds)(nil)
	_ plugin.Serializable = (*Kinds)(nil)
)

func newKinds() *Kinds { return &Kinds{entries: make(map[string]registered)} }

// Register takes spec on as name and assigns its ID and SpriteID by call order.
func (k *Kinds) Register(name string, row reflect.Type, spec kind.Spec) (kind.ID, render.SpriteID) {
	if len(k.order) == kind.MaxKinds {
		panic(fmt.Sprintf("world: cannot define kind %q: a world holds at most %d kinds", name, kind.MaxKinds))
	}
	if _, taken := k.entries[name]; taken {
		panic(fmt.Sprintf("world: kind %q is defined twice", name))
	}
	r := registered{name: name, typeID: kind.ID(len(k.order)), spriteID: k.NewSprite(), row: row}
	var positions, velocities int
	for _, c := range spec {
		switch t := c.(type) {
		case kind.Template[Position]:
			r.position, positions = t, positions+1
		case kind.Template[Velocity]:
			r.velocity, velocities = t, velocities+1
		default:
			r.comps = append(r.comps, c)
		}
	}
	if positions != 1 || velocities != 1 {
		panic(fmt.Sprintf("world: kind %q names %d Position and %d Velocity components, want one of each", name, positions, velocities))
	}
	k.order = append(k.order, name)
	k.entries[name] = r
	return r.typeID, r.spriteID
}

// NewSprite reserves an atlas slot that belongs to no kind — an overlay's, say.
func (k *Kinds) NewSprite() render.SpriteID {
	id := k.next
	k.next++
	return id
}

// SetupSystems returns no systems.
func (k *Kinds) SetupSystems() []goke.System { return nil }

// LoadComps lists every component type the defined kinds give their entities, each once.
func (k *Kinds) LoadComps() []goke.CompToken {
	var tokens []goke.CompToken
	listed := map[string]bool{}
	for _, name := range k.order {
		for _, c := range k.entries[name].comps {
			if token := c.LoadToken(); !listed[token.Name] {
				listed[token.Name] = true
				tokens = append(tokens, token)
			}
		}
	}
	return tokens
}

// Persisted returns the kind.ID to name mapping for Persistence.Save and Load.
func (k *Kinds) Persisted() []any {
	k.saved = k.order
	return []any{&k.saved}
}

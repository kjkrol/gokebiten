package plugins

import (
	"reflect"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/resource"
)

// GameCtx is the capability surface Game exposes to Plugin.Install.
type GameCtx struct {
	Resources *Resources

	ecs        *goke.ECS
	track      func(v any)
	addPending func(producer func() []goke.System)
	installed  func(name string) bool

	wrote bool
}

// NewGameCtx builds a GameCtx — called by gokebiten per Install attempt.
func NewGameCtx(resources *Resources, ecs *goke.ECS, track func(v any), addPending func(func() []goke.System), installed func(name string) bool) *GameCtx {
	return &GameCtx{Resources: resources, ecs: ecs, track: track, addPending: addPending, installed: installed}
}

// Require returns the published value of T, or an error if it isn't published yet — Install may be retried until it is.
func (c *GameCtx) Require[T resource.PluginResource]() (T, error) {
	if v, ok := c.Resources.TryGet[T](); ok {
		return v, nil
	}
	var zero T
	return zero, &NotReadyError{Reason: "requires " + reflect.TypeFor[T]().String() + ", not yet published"}
}

// RequirePlugin blocks Install until p has itself finished installing —
// Install may be retried until it has. Unlike Require, this never touches
// Resources; it's for plugin-to-plugin install ordering, not for reading data.
func (c *GameCtx) RequirePlugin(p Plugin) error {
	if c.installed(p.Name()) {
		return nil
	}
	return &NotReadyError{Reason: `requires plugin "` + p.Name() + `" to finish installing`}
}

// Provide publishes v so other plugins and the rest of the game can read it.
func (c *GameCtx) Provide[T resource.PluginResource](v T) {
	c.wrote = true
	c.Resources.Insert(v)
}

// RegComp registers ECS component type C.
func (c *GameCtx) RegComp[C any]() goke.CompID { return c.ecs.RegComp[C]() }

// UseModule registers m's per-tick systems.
func (c *GameCtx) UseModule(m goke.Module) {
	c.wrote = true
	regSys := goke.SystemFn{OnInit: func(si *goke.SysInit) { m.RegSystems(c.ecs) }}
	c.track(m)
	c.addPending(func() []goke.System { return append(m.SetupSystems(), regSys) })
}

// Setup registers providers' one-time setup systems.
func (c *GameCtx) Setup(providers ...goke.SetupProvider) {
	c.wrote = true
	for _, p := range providers {
		c.track(p)
		c.addPending(p.SetupSystems)
	}
}

// RegSys registers a single ad-hoc ECS system.
func (c *GameCtx) RegSys(factory func() goke.System) goke.Runnable {
	c.wrote = true
	return c.ecs.RegSys(factory())
}

// ECS returns the underlying goke ECS.
func (c *GameCtx) ECS() *goke.ECS { return c.ecs }

// Wrote reports whether Provide/UseModule/Setup/RegSys was called — pluginManager's retry-safety check.
func (c *GameCtx) Wrote() bool { return c.wrote }

// NotReadyError is returned by Require/RequirePlugin when the dependency isn't ready yet — pluginManager retries Install on it.
type NotReadyError struct{ Reason string }

func (e *NotReadyError) Error() string { return "gokebiten: " + e.Reason }

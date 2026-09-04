package gamecore

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin adapts a plain closure to plugins.Plugin — see gokebiten.Game.Init.
type Plugin struct {
	fn func(ctx *plugins.GameCtx) error
}

// New builds a Plugin running fn from Install.
func New(fn func(ctx *plugins.GameCtx) error) *Plugin { return &Plugin{fn: fn} }

func (p *Plugin) Name() string { return "gokebiten.game" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error { return p.fn(ctx) }

// RunPlan is a no-op — Plugin has no per-tick work of its own.
func (p *Plugin) RunPlan(goke.RunCtx, time.Duration) {}

// WithRenderer is a no-op — Plugin has no render.Renderer of its own.
func (p *Plugin) WithRenderer(render.AtlasSource) {}

// Renderer is a no-op — Plugin has no render.Renderer of its own.
func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler is a no-op — Plugin has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

package camera

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// Plugin builds a render.BasicCamera sized from world.Config if a
// world.Plugin published one, else from GameProps, and publishes it as a
// render.Camera resource. It's a plugin to resolve that world.Config
// dependency safely regardless of registration order — not to make Camera
// implementations swappable (any render.Camera can already be provided
// without this plugin).
type Plugin struct {
	viewport render.AABB

	camera *render.BasicCamera
}

var _ gokebiten.Plugin = (*Plugin)(nil)

// NewPlugin builds a camera — pass viewport to override the auto-derived full-screen one at (0,0).
func NewPlugin(viewport ...render.AABB) *Plugin {
	p := &Plugin{}
	if len(viewport) > 0 {
		p.viewport = viewport[0]
	}
	return p
}

func (p *Plugin) Name() string { return "gokebiten.camera" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	props := ctx.Resources.Get[*gokebiten.GameProps]()

	var surface plane.Space2D[uint32]
	viewport := p.viewport
	if cfg, ok := ctx.Resources.TryGet[world.Config](); ok {
		if cfg.Space.Toroidal {
			surface = plane.NewToroidal2D(cfg.Space.Width, cfg.Space.Height)
		} else {
			surface = plane.NewEuclidean2D(cfg.Space.Width, cfg.Space.Height)
		}
	} else {
		surface = plane.NewEuclidean2D(uint32(props.ScreenWidth), uint32(props.ScreenHeight))
	}
	if viewport.Equals(render.AABB{}) {
		viewport = geom.NewAABBAt(geom.NewVec[uint32](0, 0), uint32(props.ScreenWidth), uint32(props.ScreenHeight))
	}

	p.camera = render.NewBasicCamera(surface, viewport)
	ctx.Provide[render.Camera](p.camera)
	return nil
}

// Camera returns the concrete *render.BasicCamera built during Install — nil before that.
func (p *Plugin) Camera() *render.BasicCamera { return p.camera }

// RunPlan is a no-op — camera has no per-tick work of its own.
func (p *Plugin) RunPlan(goke.RunCtx, time.Duration) {}

// Renderer is a no-op — camera has no render.Renderer of its own.
func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler is a no-op — camera has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

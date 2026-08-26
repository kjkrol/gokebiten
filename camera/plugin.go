package camera

import (
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// Plugin builds a render.BasicCamera sized from world.Config if a world.Plugin published one, else from GameProps.
// Publishes itself as a render.Camera resource.
type Plugin struct {
	viewport render.AABB

	camera *render.BasicCamera
}

func NewPlugin() *Plugin { return &Plugin{} }

// WithViewport overrides the camera's initial world-space viewport instead of deriving a full-screen one at (0,0).
func (p *Plugin) WithViewport(viewport render.AABB) *Plugin {
	p.viewport = viewport
	return p
}

func (p *Plugin) Name() string { return "gokebiten.camera" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	props := ctx.Resources.GetResource[*gokebiten.GameProps]()

	var surface plane.Space2D[uint32]
	viewport := p.viewport
	if cfg, ok := ctx.Resources.TryGetResource[world.Config](); ok {
		if cfg.Toroidal {
			surface = plane.NewToroidal2D(cfg.Width, cfg.Height)
		} else {
			surface = plane.NewEuclidean2D(cfg.Width, cfg.Height)
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

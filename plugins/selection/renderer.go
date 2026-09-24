package selection

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/players"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/render"
)

// HighlightStyle draws one Selected entity's outline, given its world-space AABB.
type HighlightStyle interface {
	Draw(screen *ebiten.Image, cam camera.Camera, box camera.AABB)
}

// HighlightStyleFn adapts a plain function to HighlightStyle.
type HighlightStyleFn func(screen *ebiten.Image, cam camera.Camera, box camera.AABB)

func (f HighlightStyleFn) Draw(screen *ebiten.Image, cam camera.Camera, box camera.AABB) {
	f(screen, cam, box)
}

var _ HighlightStyle = HighlightStyleFn(nil)

var highlightColor = color.RGBA{R: 220, G: 40, B: 40, A: 255}

// DefaultHighlightStyle draws a thin red outline around box.
func DefaultHighlightStyle() HighlightStyle {
	return HighlightStyleFn(func(screen *ebiten.Image, cam camera.Camera, box camera.AABB) {
		var buf [4]camera.Quad
		for _, q := range cam.ToScreenQuads(
			float32(box.TopLeft.X), float32(box.TopLeft.Y),
			float32(box.BottomRight.X), float32(box.BottomRight.Y), buf[:0]) {
			vector.StrokeRect(screen, q.X0, q.Y0, q.X1-q.X0, q.Y1-q.Y0, 2, highlightColor, true)
		}
	})
}

var marqueeColor = color.RGBA{R: 255, G: 140, B: 0, A: 255}

// Renderer outlines every Selected entity and draws the marquee of each local player's drag in progress.
type Renderer struct {
	players *players.Plugin
	camera  camera.Camera
	style   HighlightStyle

	query    *goke.Query
	base     goke.Comp[world.Base]
	marks    goke.Comp[plugin.Tags[Family]]
	selected plugin.Tag[Family]
}

var _ render.Renderer = (*Renderer)(nil)

// NewRenderer builds a Renderer with DefaultHighlightStyle; pl's local players' drags are drawn.
func NewRenderer(cam camera.Camera, pl *players.Plugin, selected plugin.Tag[Family]) *Renderer {
	return &Renderer{players: pl, camera: cam, style: DefaultHighlightStyle(), selected: selected}
}

// WithStyle overrides how the highlight is drawn — the escape hatch for a custom HighlightStyle.
func (r *Renderer) WithStyle(style HighlightStyle) *Renderer {
	r.style = style
	return r
}

func (r *Renderer) Init(si *goke.SysInit) {
	r.query = si.NewQueryBuilder(&r.base, &r.marks).Build()
}

func (r *Renderer) Draw(screen *ebiten.Image) {
	r.query.All()
	for r.query.Next() {
		cursor := r.query.Cursor()
		bases := r.base.Slice(cursor)
		marks := r.marks.Slice(cursor)
		for i := range cursor.IDs {
			if marks[i].Has(r.selected) {
				r.style.Draw(screen, r.camera, bases[i].Pos.AABB.AABB)
			}
		}
	}

	if r.players == nil {
		return
	}
	for _, pl := range r.players.Locals() {
		start, current, dragging := pl.DragBox()
		if !dragging {
			continue
		}
		x0, y0 := float32(start.X), float32(start.Y)
		x1, y1 := float32(current.X), float32(current.Y)
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		if y1 < y0 {
			y0, y1 = y1, y0
		}
		vector.StrokeRect(screen, x0, y0, x1-x0, y1-y0, 1, marqueeColor, true)
	}
}

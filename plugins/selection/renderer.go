package selection

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/plugin"
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

// Renderer outlines every Selected entity; the marquee of a drag is the players plugin's to draw.
type Renderer struct {
	camera camera.Camera
	style  HighlightStyle

	query    *goke.Query
	base     goke.Comp[world.Base]
	marks    goke.Comp[plugin.Tags[Family]]
	selected plugin.Tag[Family]
}

var _ render.Renderer = (*Renderer)(nil)

// NewRenderer builds a Renderer with DefaultHighlightStyle.
func NewRenderer(cam camera.Camera, selected plugin.Tag[Family]) *Renderer {
	return &Renderer{camera: cam, style: DefaultHighlightStyle(), selected: selected}
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
}

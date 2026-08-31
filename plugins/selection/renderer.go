package selection

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// HighlightStyle draws one Selected entity's outline, given its world-space AABB.
type HighlightStyle interface {
	Draw(screen *ebiten.Image, camera render.Camera, box render.AABB)
}

// HighlightStyleFn adapts a plain function to HighlightStyle.
type HighlightStyleFn func(screen *ebiten.Image, camera render.Camera, box render.AABB)

func (f HighlightStyleFn) Draw(screen *ebiten.Image, camera render.Camera, box render.AABB) {
	f(screen, camera, box)
}

var _ HighlightStyle = HighlightStyleFn(nil)

var highlightColor = color.RGBA{R: 220, G: 40, B: 40, A: 255}

// DefaultHighlightStyle draws a thin red outline around box.
func DefaultHighlightStyle() HighlightStyle {
	return HighlightStyleFn(func(screen *ebiten.Image, camera render.Camera, box render.AABB) {
		x0, y0 := camera.ToScreen(float32(box.TopLeft.X), float32(box.TopLeft.Y))
		x1, y1 := camera.ToScreen(float32(box.BottomRight.X), float32(box.BottomRight.Y))
		vector.StrokeRect(screen, x0, y0, x1-x0, y1-y0, 2, highlightColor, true)
	})
}

var marqueeColor = color.RGBA{R: 255, G: 140, B: 0, A: 255}

// Renderer draws a thin outline around every Selected entity, plus a live
// marquee rectangle while a drag-select is in progress — register alongside
// your other layers.
type Renderer struct {
	state  *State
	camera render.Camera
	style  HighlightStyle

	query *goke.Query
	pos   goke.Comp[world.Position]
}

var _ render.Renderer = (*Renderer)(nil)

// NewRenderer builds a Renderer over state's live selection/drag state, with DefaultHighlightStyle — override via WithStyle.
func NewRenderer(state *State) *Renderer {
	return &Renderer{state: state, style: DefaultHighlightStyle()}
}

// WithStyle overrides how the highlight is drawn — the escape hatch for a custom HighlightStyle.
func (r *Renderer) WithStyle(style HighlightStyle) *Renderer {
	r.style = style
	return r
}

// BindCamera attaches camera — Draw needs it, so call this before the first Draw.
func (r *Renderer) BindCamera(camera render.Camera) { r.camera = camera }

func (r *Renderer) Init(si *goke.SysInit) {
	r.query = si.NewQueryBuilder(&r.pos).Include(goke.Include[Selected]()).Build()
}

func (r *Renderer) Draw(screen *ebiten.Image) {
	r.query.All()
	for r.query.Next() {
		cursor := r.query.Cursor()
		positions := r.pos.Slice(cursor)
		for i := range cursor.IDs {
			r.style.Draw(screen, r.camera, positions[i].AABB.AABB)
		}
	}

	if start, current, dragging := r.state.DragBox(); dragging {
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

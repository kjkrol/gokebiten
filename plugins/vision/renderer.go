package vision

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// ConeStyle draws one entity's view, given the fan already projected to screen
// space: pts[0] is the observer, the rest its boundary in order.
type ConeStyle interface {
	Draw(screen *ebiten.Image, pts []ebiten.Vertex)
}

// ConeStyleFn adapts a plain function to ConeStyle.
type ConeStyleFn func(screen *ebiten.Image, pts []ebiten.Vertex)

func (f ConeStyleFn) Draw(screen *ebiten.Image, pts []ebiten.Vertex) { f(screen, pts) }

var _ ConeStyle = ConeStyleFn(nil)

var coneColor = color.RGBA{R: 255, G: 220, B: 90, A: 160}

// DefaultConeStyle strokes the boundary of the lit region and leaves the inside clear.
func DefaultConeStyle() ConeStyle {
	return ConeStyleFn(func(screen *ebiten.Image, pts []ebiten.Vertex) {
		var path vector.Path
		path.MoveTo(pts[0].DstX, pts[0].DstY)
		for _, p := range pts[1:] {
			path.LineTo(p.DstX, p.DstY)
		}
		path.Close()
		vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 1}, &vector.DrawPathOptions{
			ColorScale: colorScaleOf(coneColor),
			AntiAlias:  true,
		})
	})
}

func colorScaleOf(c color.RGBA) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	return cs
}

var _ render.Renderer = (*Renderer)(nil)

// Renderer draws the view of every entity carrying SightOutline.
type Renderer struct {
	camera camera.Camera
	space  *aabbworld.Space
	style  ConeStyle

	worldW, worldH float32
	wraps          bool

	query *goke.Query
	base  goke.Comp[world.Base]
	sight goke.Comp[Sight]
	out   goke.Comp[SightOutline]

	pts []ebiten.Vertex // rebuilt per entity, kept to stay off the heap
}

// NewRenderer builds a Renderer with DefaultConeStyle, wrapping cones at the edges of space.
func NewRenderer(cam camera.Camera, space *aabbworld.Space) *Renderer {
	w, h, edges := space.Bounds()
	return &Renderer{
		camera: cam, space: space, style: DefaultConeStyle(),
		worldW: float32(w), worldH: float32(h), wraps: edges&aabbworld.Torus != 0,
	}
}

// Style reports how cones are currently drawn.
func (r *Renderer) Style() ConeStyle { return r.style }

// WithStyle replaces how each cone is drawn.
func (r *Renderer) WithStyle(style ConeStyle) *Renderer {
	r.style = style
	return r
}

func (r *Renderer) Init(si *goke.SysInit) {
	r.query = si.NewQueryBuilder(&r.base, &r.sight, &r.out).Build()
}

func (r *Renderer) Draw(screen *ebiten.Image) {
	r.query.All()
	for r.query.Next() {
		cursor := r.query.Cursor()
		bases := r.base.Slice(cursor)
		sights := r.sight.Slice(cursor)
		outlines := r.out.Slice(cursor)

		for i := range cursor.IDs {
			if outlines[i].Count < 2 || !r.camera.Visible(bases[i].Pos.AABB.AABB) {
				continue
			}
			r.drawCone(screen, &bases[i].Pos, &sights[i], &outlines[i])
		}
	}
}

// drawCone draws one entity's view once per image of the world it reaches into.
func (r *Renderer) drawCone(screen *ebiten.Image, pos *world.Position, s *Sight, o *SightOutline) {
	ox, oy := centreOf(pos)
	sx, sy := r.camera.ToScreen(float32(ox), float32(oy))

	if !r.wraps {
		r.style.Draw(screen, r.fan(sx, sy, s, o))
		return
	}

	zoom := r.camera.Zoom()
	box := r.space.WrapAABB(r.coneBox(ox, oy, s.Radius))
	render.VisitWrapImages(box, r.worldW, r.worldH, func(_ geom.AABB, dx, dy float32) bool {
		r.style.Draw(screen, r.fan(sx+dx*zoom, sy+dy*zoom, s, o))
		return true
	})
}

// coneBox is the square a cone covers, clamped to the size of the world.
func (r *Renderer) coneBox(ox, oy, radius float64) geom.AABB {
	w, h := float64(r.worldW), float64(r.worldH)
	return geom.NewAABBAt(
		geom.NewVec(ox-radius, oy-radius),
		math.Min(2*radius, w), math.Min(2*radius, h),
	)
}

// fan rebuilds the boundary around an already-projected anchor from the stored reaches.
func (r *Renderer) fan(sx, sy float32, s *Sight, o *SightOutline) []ebiten.Vertex {
	facing := math.Atan2(s.Facing.Y, s.Facing.X)
	step := 2 * s.HalfAngle / float64(o.Count-1)
	zoom := r.camera.Zoom()

	r.pts = append(r.pts[:0], ebiten.Vertex{DstX: sx, DstY: sy})
	for i := range int(o.Count) {
		a := facing - s.HalfAngle + float64(i)*step
		d := float64(o.Depths[i])
		r.pts = append(r.pts, ebiten.Vertex{
			DstX: sx + float32(d*math.Cos(a))*zoom,
			DstY: sy + float32(d*math.Sin(a))*zoom,
		})
	}
	return r.pts
}

func centreOf(p *world.Position) (float64, float64) {
	box := p.AABB.AABB
	return (float64(box.TopLeft.X) + float64(box.BottomRight.X)) / 2,
		(float64(box.TopLeft.Y) + float64(box.BottomRight.Y)) / 2
}

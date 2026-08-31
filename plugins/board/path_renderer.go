package board

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/selection"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

// PathStyle draws one entity's remaining route, given as consecutive
// world-space waypoint centers (points[0] is the entity's current position).
type PathStyle interface {
	Draw(screen *ebiten.Image, camera render.Camera, points []geom.Vec[float64])
}

// PathStyleFn adapts a plain function to PathStyle.
type PathStyleFn func(screen *ebiten.Image, camera render.Camera, points []geom.Vec[float64])

func (f PathStyleFn) Draw(screen *ebiten.Image, camera render.Camera, points []geom.Vec[float64]) {
	f(screen, camera, points)
}

var _ PathStyle = PathStyleFn(nil)

var pathColor = color.RGBA{R: 255, G: 140, B: 0, A: 255}

// DefaultPathStyle draws an orange polyline through points with a dot marking the final destination.
func DefaultPathStyle() PathStyle {
	return PathStyleFn(func(screen *ebiten.Image, camera render.Camera, points []geom.Vec[float64]) {
		for i := 0; i+1 < len(points); i++ {
			x0, y0 := camera.ToScreen(float32(points[i].X), float32(points[i].Y))
			x1, y1 := camera.ToScreen(float32(points[i+1].X), float32(points[i+1].Y))
			vector.StrokeLine(screen, x0, y0, x1, y1, 2, pathColor, true)
		}
		if len(points) == 0 {
			return
		}
		last := points[len(points)-1]
		x, y := camera.ToScreen(float32(last.X), float32(last.Y))
		vector.FillCircle(screen, x, y, 4, pathColor, true)
	})
}

// pathPoints returns the world-space centers PathRenderer should draw
// through: the entity's current position, then every remaining Path step,
// ending at target.
func pathPoints(grid Grid, pos world.Position, path Path, target CellID) []geom.Vec[float64] {
	points := []geom.Vec[float64]{center(pos)}
	for s := path.Index; s < path.Length; s++ {
		points = append(points, grid.CellCenter(path.Steps[s]))
	}
	if path.Length == 0 || path.Index >= path.Length {
		points = append(points, grid.CellCenter(target))
	}
	return points
}

// PathRenderer draws the remaining route for every selection.Selected
// entity — register alongside Renderer for a selection/debug overlay.
type PathRenderer struct {
	grid   Grid
	camera render.Camera
	style  PathStyle

	query  *goke.Query
	pos    goke.Comp[world.Position]
	path   goke.Comp[Path]
	moveTo goke.Comp[MoveTo]
}

var _ render.Renderer = (*PathRenderer)(nil)

// NewPathRenderer builds a PathRenderer over grid, with DefaultPathStyle — override via WithStyle.
func NewPathRenderer(grid Grid) *PathRenderer {
	return &PathRenderer{grid: grid, style: DefaultPathStyle()}
}

// WithStyle overrides how the route is drawn — the escape hatch for a custom PathStyle.
func (r *PathRenderer) WithStyle(style PathStyle) *PathRenderer {
	r.style = style
	return r
}

// BindCamera attaches camera — Draw needs it, so call this before the first Draw.
func (r *PathRenderer) BindCamera(camera render.Camera) { r.camera = camera }

func (r *PathRenderer) Init(si *goke.SysInit) {
	r.query = si.NewQueryBuilder(&r.pos, &r.path, &r.moveTo).
		Include(goke.Include[selection.Selected]()).
		Build()
}

func (r *PathRenderer) Draw(screen *ebiten.Image) {
	r.query.All()
	for r.query.Next() {
		cursor := r.query.Cursor()
		positions := r.pos.Slice(cursor)
		paths := r.path.Slice(cursor)
		moveTos := r.moveTo.Slice(cursor)
		for i := range cursor.IDs {
			points := pathPoints(r.grid, positions[i], paths[i], moveTos[i].Target)
			r.style.Draw(screen, r.camera, points)
		}
	}
}

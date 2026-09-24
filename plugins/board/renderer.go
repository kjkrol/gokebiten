package board

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/render"
)

var colorGridLine = color.RGBA{R: 20, G: 20, B: 20, A: 120}

// RenderState is the board renderer's live display toggles.
type RenderState struct {
	ShowGridLines bool
}

// ToggleShowGridLines flips whether grid lines are drawn.
func (r *RenderState) ToggleShowGridLines() { r.ShowGridLines = !r.ShowGridLines }

// Renderer draws Board's cells — register it before the entities layer in Game.Layers so terrain
// sits underneath, or hand it to render.NewSorted, where it submits each cell as a quad at the
// cell's altitude and depth (no grid lines there).
type Renderer struct {
	board     *Board
	camera    camera.Camera
	atlas     render.AtlasSource
	cellW     float64
	cellH     float64
	state     *RenderState
	batch     *render.QuadBatch
	gridLines []gridLine
	outline   []geom.Vec
	visited   map[CellID]struct{}
	// relief draws the sides of raised ground and of tall kinds: an isometric camera's view.
	relief bool
}

// Shades of a block's faces against its top: the side facing down-right and the one facing
// down-left, as if lit from the upper left.
const (
	shadeRight = 0.72
	shadeLeft  = 0.55
)

type gridLine struct{ x0, y0, x1, y1 float32 }

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

var _ render.Submitter = (*Renderer)(nil)

func newRenderer(cam camera.Camera, board *Board, atlas render.AtlasSource, state *RenderState) *Renderer {
	w, h := board.CellBounds()
	_, iso := cam.Projection().(camera.Isometric)
	return &Renderer{board: board, camera: cam, atlas: atlas, cellW: w, cellH: h, state: state,
		batch: render.NewQuadBatch(atlas, cam), visited: map[CellID]struct{}{}, relief: iso}
}

func (l *Renderer) Init(*goke.SysInit) {}

func (l *Renderer) Draw(screen *ebiten.Image) {
	l.batch.Reset()
	l.gridLines = l.gridLines[:0]
	l.eachVisible(l.drawCell)
	l.batch.Flush(screen)

	for _, gl := range l.gridLines {
		vector.StrokeLine(screen, gl.x0, gl.y0, gl.x1, gl.y1, 1, colorGridLine, false)
	}
}

// Submit hands every visible cell to sink at the depth of its centre, so the entities standing on
// it follow it: its top at its altitude plus its kind's Height, and, through an isometric camera,
// the two faces towards the viewer wherever the ground drops to a neighbour or the kind stands tall.
func (l *Renderer) Submit(sink *render.Sink) {
	l.eachVisible(func(c CellID) {
		center := l.board.CellCenter(c)
		kind := l.board.Kind(c)
		alt := float32(l.board.Altitude(c))
		top := alt + float32(kind.Height)
		x0, y0 := float32(center.X-l.cellW/2), float32(center.Y-l.cellH/2)
		x1, y1 := float32(center.X+l.cellW/2), float32(center.Y+l.cellH/2)
		depth := l.camera.Depth(float32(center.X), float32(center.Y), alt)
		if l.relief {
			// the face along x = x1 shows down to the neighbour across it, the one along y = y1 likewise
			if foot := l.neighbourAltitude(center.X+l.cellW, center.Y); foot < top {
				sink.Shaded(depth, l.atlas, kind.SpriteID, l.face(x1, y0, x1, y1, top, foot), shadeRight)
			}
			if foot := l.neighbourAltitude(center.X, center.Y+l.cellH); foot < top {
				sink.Shaded(depth, l.atlas, kind.SpriteID, l.face(x0, y1, x1, y1, top, foot), shadeLeft)
			}
		}
		sink.Quad(depth, l.atlas, kind.SpriteID, l.corners(x0, y0, x1, y1, top))
	})
}

// neighbourAltitude is the ground level of the cell at p, the sea level 0 off the board.
func (l *Renderer) neighbourAltitude(x, y float64) float32 {
	c, ok := l.board.CellAt(geom.NewVec(x, y))
	if !ok {
		return 0
	}
	return float32(l.board.Altitude(c))
}

// corners projects the four corners of a world box at height z.
func (l *Renderer) corners(x0, y0, x1, y1, z float32) render.Corners {
	var out render.Corners
	for i, p := range [4][2]float32{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}} {
		out[i][0], out[i][1] = l.camera.Project(p[0], p[1], z)
	}
	return out
}

// face projects a vertical wall from the edge (ax, ay)-(bx, by) between heights top and foot.
func (l *Renderer) face(ax, ay, bx, by, top, foot float32) render.Corners {
	var out render.Corners
	out[0][0], out[0][1] = l.camera.Project(ax, ay, top)
	out[1][0], out[1][1] = l.camera.Project(bx, by, top)
	out[2][0], out[2][1] = l.camera.Project(ax, ay, foot)
	out[3][0], out[3][1] = l.camera.Project(bx, by, foot)
	return out
}

// eachVisible calls fn once for every cell under the camera's bounds.
func (l *Renderer) eachVisible(fn func(c CellID)) {
	step := min(l.cellW, l.cellH) / 2
	if step <= 0 {
		step = 1
	}
	bounds := l.camera.Bounds()
	clear(l.visited)
	for y := float64(bounds.TopLeft.Y); y < float64(bounds.BottomRight.Y)+step; y += step {
		for x := float64(bounds.TopLeft.X); x < float64(bounds.BottomRight.X)+step; x += step {
			c, ok := l.board.CellAt(geom.NewVec(x, y))
			if !ok {
				continue
			}
			if _, seen := l.visited[c]; seen {
				continue
			}
			l.visited[c] = struct{}{}
			fn(c)
		}
	}
}

func (l *Renderer) drawCell(c CellID) {
	center := l.board.CellCenter(c)
	x0, y0 := center.X-l.cellW/2, center.Y-l.cellH/2
	x1, y1 := center.X+l.cellW/2, center.Y+l.cellH/2

	l.batch.AppendQuad(float32(x0), float32(y0), float32(x1), float32(y1), l.board.Kind(c).SpriteID)

	if l.state.ShowGridLines {
		l.outline = l.board.CellOutline(c, l.outline[:0])
		for i, p := range l.outline {
			q := l.outline[(i+1)%len(l.outline)]
			ax, ay := l.camera.ToScreen(float32(p.X), float32(p.Y))
			bx, by := l.camera.ToScreen(float32(q.X), float32(q.Y))
			if reach := float32(l.cellW+l.cellH) * l.camera.Zoom(); abs32(bx-ax) > reach || abs32(by-ay) > reach {
				continue // the edge straddles a wrap seam; its images are drawn by the cells either side
			}
			l.gridLines = append(l.gridLines, gridLine{ax, ay, bx, by})
		}
	}
}

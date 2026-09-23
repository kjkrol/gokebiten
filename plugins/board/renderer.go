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

// Renderer draws Board's cells — register it before the entities layer in
// Game.Layers so terrain sits underneath.
type Renderer struct {
	board     *Board
	camera    camera.Camera
	cellW     float64
	cellH     float64
	state     *RenderState
	batch     *render.QuadBatch
	gridLines []gridLine
	outline   []geom.Vec
}

type gridLine struct{ x0, y0, x1, y1 float32 }

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

var _ render.Renderer = (*Renderer)(nil)

func newRenderer(cam camera.Camera, board *Board, atlas render.AtlasSource, state *RenderState) *Renderer {
	w, h := board.CellBounds()
	return &Renderer{board: board, camera: cam, cellW: w, cellH: h, state: state, batch: render.NewQuadBatch(atlas, cam)}
}

func (l *Renderer) Init(*goke.SysInit) {}

func (l *Renderer) Draw(screen *ebiten.Image) {
	l.batch.Reset()
	l.gridLines = l.gridLines[:0]
	step := min(l.cellW, l.cellH) / 2
	if step <= 0 {
		step = 1
	}
	bounds := l.camera.Bounds()

	visited := make(map[CellID]struct{})
	for y := float64(bounds.TopLeft.Y); y < float64(bounds.BottomRight.Y)+step; y += step {
		for x := float64(bounds.TopLeft.X); x < float64(bounds.BottomRight.X)+step; x += step {
			c, ok := l.board.CellAt(geom.NewVec(x, y))
			if !ok {
				continue
			}
			if _, seen := visited[c]; seen {
				continue
			}
			visited[c] = struct{}{}
			l.drawCell(c)
		}
	}
	l.batch.Flush(screen)

	for _, gl := range l.gridLines {
		vector.StrokeLine(screen, gl.x0, gl.y0, gl.x1, gl.y1, 1, colorGridLine, false)
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

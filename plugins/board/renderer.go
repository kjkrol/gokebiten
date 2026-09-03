package board

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

var colorGridLine = color.RGBA{R: 20, G: 20, B: 20, A: 120}

// RenderState is the board renderer's live display toggles — published to
// Resources by Plugin.WithRenderer, so a game can flip them directly (e.g.
// bind a key to renderState.ShowGridLines = !renderState.ShowGridLines).
type RenderState struct {
	ShowGridLines bool
}

// Renderer draws Board's cells — register it before the entities layer in
// Game.Layers so terrain sits underneath.
type Renderer struct {
	board     *Board
	camera    render.Camera
	cellSize  float32
	state     *RenderState
	batch     *render.QuadBatch
	gridLines []gridLine
}

// gridLine is one cell's screen-space grid-line rect, drawn after the
// batched fill so it isn't painted over by it.
type gridLine struct{ x0, y0, x1, y1 float32 }

var _ render.Renderer = (*Renderer)(nil)

func newRenderer(board *Board, atlas render.AtlasSource, state *RenderState) *Renderer {
	return &Renderer{board: board, cellSize: board.CellSpan(), state: state, batch: render.NewQuadBatch(atlas)}
}

// BindCamera attaches camera — Draw needs it, so call this before the first Draw.
func (l *Renderer) BindCamera(camera render.Camera) { l.camera = camera; l.batch.BindCamera(camera) }

func (l *Renderer) Init(*goke.SysInit) {}

func (l *Renderer) Draw(screen *ebiten.Image) {
	l.batch.Reset()
	l.gridLines = l.gridLines[:0]
	step := float64(l.cellSize)
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
		vector.StrokeRect(screen, gl.x0, gl.y0, gl.x1-gl.x0, gl.y1-gl.y0, 1, colorGridLine, false)
	}
}

func (l *Renderer) drawCell(c CellID) {
	center := l.board.CellCenter(c)
	half := float64(l.cellSize) / 2
	x0, y0 := center.X-half, center.Y-half
	x1, y1 := center.X+half, center.Y+half

	l.batch.AppendQuad(float32(x0), float32(y0), float32(x1), float32(y1), l.board.Kind(c).SpriteID)

	if l.state.ShowGridLines {
		sx0, sy0 := l.camera.ToScreen(float32(x0), float32(y0))
		sx1, sy1 := l.camera.ToScreen(float32(x1), float32(y1))
		l.gridLines = append(l.gridLines, gridLine{sx0, sy0, sx1, sy1})
	}
}

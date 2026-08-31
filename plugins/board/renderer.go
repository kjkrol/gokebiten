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

// CellStyle picks a cell's fill color from its terrain — the game decides
// the palette, Renderer only asks.
type CellStyle func(c CellID, cost float64, passable bool) color.RGBA

// Renderer draws Board's cells — register it before the entities layer in
// Game.Layers so terrain sits underneath.
type Renderer struct {
	board    *Board
	camera   render.Camera
	cellSize float32
	style    CellStyle
	showGrid bool
}

var _ render.Renderer = (*Renderer)(nil)

func newRenderer(board *Board, cellSize float32, style CellStyle) *Renderer {
	return &Renderer{board: board, cellSize: cellSize, style: style, showGrid: true}
}

// BindCamera attaches camera — Draw needs it, so call this before the first Draw.
func (l *Renderer) BindCamera(camera render.Camera) { l.camera = camera }

// SetShowGridLines toggles lines between cells — cell fill always draws.
func (l *Renderer) SetShowGridLines(show bool) { l.showGrid = show }

func (l *Renderer) Init(*goke.SysInit) {}

func (l *Renderer) Draw(screen *ebiten.Image) {
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
			l.drawCell(screen, c)
		}
	}
}

func (l *Renderer) drawCell(screen *ebiten.Image, c CellID) {
	center := l.board.CellCenter(c)
	half := float64(l.cellSize) / 2
	sx0, sy0 := l.camera.ToScreen(float32(center.X-half), float32(center.Y-half))
	sx1, sy1 := l.camera.ToScreen(float32(center.X+half), float32(center.Y+half))

	cost, passable := l.board.MovementCost(c)
	fill := l.style(c, cost, passable)

	vector.FillRect(screen, sx0, sy0, sx1-sx0, sy1-sy0, fill, false)
	if l.showGrid {
		vector.StrokeRect(screen, sx0, sy0, sx1-sx0, sy1-sy0, 1, colorGridLine, false)
	}
}

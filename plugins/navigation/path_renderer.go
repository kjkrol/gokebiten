package navigation

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/selection"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
)

func pathCells(cell board.Cell, mt MoveOrder) []board.CellID {
	cells := []board.CellID{cell.ID}
	for s := mt.Path.Index; s < mt.Path.Length; s++ {
		if step := mt.Path.Steps[s]; step != cells[len(cells)-1] {
			cells = append(cells, step)
		}
	}
	if mt.Path.Length == 0 || mt.Path.Index >= mt.Path.Length {
		if mt.Target != cells[len(cells)-1] {
			cells = append(cells, mt.Target)
		}
	}
	return cells
}

type PathRenderer struct {
	grid    board.Grid
	sprites PathSprites
	batch   *render.QuadBatch
	space   *gokg.Space

	query *goke.Query
	pos   goke.Comp[world.Position]
	vel   goke.Comp[world.Velocity]
	order goke.Comp[MoveOrder]
}

var _ render.Renderer = (*PathRenderer)(nil)

func NewPathRenderer(grid board.Grid, atlas render.AtlasSource, sprites PathSprites) *PathRenderer {
	return &PathRenderer{grid: grid, sprites: sprites, batch: render.NewQuadBatch(atlas)}
}

func (r *PathRenderer) BindSpace(space *gokg.Space) { r.space = space }

func (r *PathRenderer) BindCamera(camera render.Camera) { r.batch.BindCamera(camera) }

func (r *PathRenderer) Init(si *goke.SysInit) {
	r.query = si.NewQueryBuilder(&r.pos, &r.vel, &r.order).
		Include(goke.Include[selection.Selected]()).
		Build()
}

func (r *PathRenderer) Draw(screen *ebiten.Image) {
	r.batch.Reset()
	if r.space != nil {
		r.query.All()
		for r.query.Next() {
			cursor := r.query.Cursor()
			positions := r.pos.Slice(cursor)
			velocities := r.vel.Slice(cursor)
			orders := r.order.Slice(cursor)
			for i := range cursor.IDs {
				center := board.Center(positions[i])
				actual, ok := r.grid.CellAt(center)
				if !ok {
					continue
				}
				r.drawPath(center, velocities[i].Dir, pathCells(board.Cell{ID: actual}, orders[i]))
			}
		}
	}
	r.batch.Flush(screen)
}

func (r *PathRenderer) drawPath(entityCenter, travel geom.Vec[float64], cells []board.CellID) {
	for i, c := range cells {
		center := r.grid.CellCenter(c)

		if i > 0 {
			dirBack := directionBetween(center, r.grid.CellCenter(cells[i-1]), r.space.Width, r.space.Height, r.space.Toroidal)
			r.appendCellSprite(c, r.sprites.spoke(dirBack))
		}

		if i == len(cells)-1 {
			r.appendCellSprite(c, r.sprites.Dot)
			continue
		}

		if i == 0 && hasPassedCenter(center, entityCenter, travel, r.space.Width, r.space.Height, r.space.Toroidal) {
			continue
		}

		dirOut := directionBetween(center, r.grid.CellCenter(cells[i+1]), r.space.Width, r.space.Height, r.space.Toroidal)
		r.appendCellSprite(c, r.sprites.spoke(dirOut))
	}
}

func (r *PathRenderer) appendCellSprite(c board.CellID, sprite render.SpriteID) {
	center := r.grid.CellCenter(c)
	half := float64(r.grid.CellSpan()) / 2
	r.batch.AppendQuad(float32(center.X-half), float32(center.Y-half), float32(center.X+half), float32(center.Y+half), sprite)
}

func hasPassedCenter(cellCenter, entityCenter, travel geom.Vec[float64], width, height uint32, toroidal bool) bool {
	ex := shortestAxisDelta(cellCenter.X, entityCenter.X, width, toroidal)
	ey := shortestAxisDelta(cellCenter.Y, entityCenter.Y, height, toroidal)
	return ex*travel.X+ey*travel.Y > 0
}

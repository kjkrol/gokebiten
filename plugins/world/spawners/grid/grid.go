package grid

import (
	"math"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

var _ world.Placement = (*GridPlacement)(nil)

// GridPlacement arranges entities on a regular grid spanning [0,Width)×[0,Height).
type GridPlacement struct {
	Width, Height uint32
	EntitySize    uint32
}

func NewGridPlacement(width, height, entitySize uint32) *GridPlacement {
	return &GridPlacement{Width: width, Height: height, EntitySize: entitySize}
}

func (p *GridPlacement) Place(index, count int) world.Position {
	cols := uint32(math.Ceil(math.Sqrt(float64(count))))
	row := uint32(index) / cols
	col := uint32(index) % cols

	cellWidth := p.Width / cols
	cellHeight := p.Height / cols

	x := (col * cellWidth) + (cellWidth / 2) - (p.EntitySize / 2)
	y := (row * cellHeight) + (cellHeight / 2) - (p.EntitySize / 2)

	return world.Position{AABB: plane.NewAABB(geom.NewVec(x, y), p.EntitySize, p.EntitySize)}
}

// Components and Init are no-ops: Place already supplies everything this
// type contributes. They exist so a GridPlacement can be passed directly as
// Populate's first populator, alongside plain EntityExtras.
func (p *GridPlacement) Components() []goke.Addable             { return nil }
func (p *GridPlacement) Init(*goke.Cursor, int, int, uid.UID64) {}

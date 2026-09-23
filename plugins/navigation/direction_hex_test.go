package navigation

import (
	"testing"

	"github.com/kjkrol/gram/plugins/board"
)

func TestDirectionBetween_HexNeighboursLandOnSixtyDegreeSteps(t *testing.T) {
	g := board.DefaultGrids{}.Hex(4, 4, 20)
	from, _ := g.CellIndex(1, 1)
	for _, c := range []struct {
		q, r uint32
		want Direction
	}{
		{2, 1, DirectionAt(0)}, {0, 1, DirectionAt(180)},
		{2, 0, DirectionAt(60)}, {1, 0, DirectionAt(120)},
		{0, 2, DirectionAt(240)}, {1, 2, DirectionAt(300)},
	} {
		to, _ := g.CellIndex(c.q, c.r)
		if got := directionBetween(g.CellCenter(from), g.CellCenter(to), 1000, 1000, 0); got != c.want {
			t.Errorf("to (%d,%d): direction %d (%v°), want %d (%v°)", c.q, c.r, got, got.Angle(), c.want, c.want.Angle())
		}
	}
	if opposite(DirectionAt(60)) != DirectionAt(240) {
		t.Errorf("opposite of 60° is %v°, want 240°", opposite(DirectionAt(60)).Angle())
	}
}

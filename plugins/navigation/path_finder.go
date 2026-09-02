package navigation

import (
	"github.com/kjkrol/astar"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/uid"
)

// PathFinder computes routes over one grid, reusing its A* solver across
// calls — build once and share across systems.
type PathFinder struct {
	grid   board.Grid
	solver *astar.Solver[board.CellID]
}

// NewPathFinder builds a PathFinder over grid, allocating its solver once for reuse across FindPath calls.
func NewPathFinder(grid board.Grid) *PathFinder {
	return &PathFinder{
		grid:   grid,
		solver: astar.New[board.CellID](func(a, b board.CellID) float64 { return grid.Distance(a, b) }),
	}
}

// Grid returns the grid this PathFinder was built over.
func (p *PathFinder) Grid() board.Grid { return p.grid }

// FindPath computes a route from 'from' toward 'to' for entity, respecting terrain/occupancy — ok=false if unreachable.
func (p *PathFinder) FindPath(terrain board.Terrain, occupancy board.Occupancy, entity uid.UID64, from, to board.CellID) (Path, bool) {
	return findPath(p.solver, p.grid, terrain, occupancy, entity, from, to)
}

func findPath(solver *astar.Solver[board.CellID], grid board.Grid, terrain board.Terrain, occupancy board.Occupancy, entity uid.UID64, from, to board.CellID) (Path, bool) {
	full := solver.Solve(from, to, transitionsFor(grid, terrain, occupancy, entity))
	if len(full) < 2 {
		return Path{}, false
	}
	steps := full[1:]
	n := min(len(steps), MaxPathLength)
	var p Path
	copy(p.Steps[:], steps[:n])
	p.Length = uint16(n)
	return p, true
}

// transitionsFor adapts Grid+Terrain+Occupancy into astar's Transitions shape for one entity's Solve call.
func transitionsFor(grid board.Grid, terrain board.Terrain, occupancy board.Occupancy, entity uid.UID64) astar.Transitions[board.CellID] {
	return func(from, prev board.CellID, buf []astar.Transition[board.CellID]) []astar.Transition[board.CellID] {
		buf = buf[:0]
		for _, n := range grid.Neighbors(from) {
			if n == prev {
				continue
			}
			kind := terrain.Kind(n)
			if !kind.Passable || !occupancy.CanEnter(n, entity) {
				continue
			}
			if c1, c2, ok := grid.DiagonalNeighbors(from, n); ok {
				if !terrain.Kind(c1).Passable || !terrain.Kind(c2).Passable {
					continue
				}
			}
			buf = append(buf, astar.Transition[board.CellID]{To: n, Cost: kind.Cost * grid.NeighborCost(from, n)})
		}
		return buf
	}
}

package board

import (
	"github.com/kjkrol/astar"
	"github.com/kjkrol/uid"
)

// PathFinder computes routes over one grid, reusing its A* solver across
// calls — build once (typically one per board.Plugin) and share it with
// every system that paths over the same grid.
type PathFinder struct {
	grid   Grid
	solver *astar.Solver[CellID]
}

// NewPathFinder builds a PathFinder over grid, allocating its solver once for reuse across FindPath calls.
func NewPathFinder(grid Grid) *PathFinder {
	return &PathFinder{
		grid:   grid,
		solver: astar.New[CellID](func(a, b CellID) float64 { return grid.Distance(a, b) }),
	}
}

// Grid returns the grid this PathFinder was built over.
func (p *PathFinder) Grid() Grid { return p.grid }

// FindPath computes a route from 'from' toward 'to' for entity, respecting terrain/occupancy — ok=false if unreachable.
func (p *PathFinder) FindPath(terrain Terrain, occupancy Occupancy, entity uid.UID64, from, to CellID) (Path, bool) {
	return findPath(p.solver, p.grid, terrain, occupancy, entity, from, to)
}

func findPath(solver *astar.Solver[CellID], grid Grid, terrain Terrain, occupancy Occupancy, entity uid.UID64, from, to CellID) (Path, bool) {
	full := solver.Solve(from, to, transitionsFor(grid, terrain, occupancy, entity))
	if len(full) < 2 {
		return Path{}, false
	}
	steps := full[1:]
	n := len(steps)
	if n > MaxPathLength {
		n = MaxPathLength
	}
	var p Path
	copy(p.Steps[:], steps[:n])
	p.Length = uint16(n)
	return p, true
}

// transitionsFor adapts Grid+Terrain+Occupancy into astar's Transitions shape for one entity's Solve call.
func transitionsFor(grid Grid, terrain Terrain, occupancy Occupancy, entity uid.UID64) astar.Transitions[CellID] {
	return func(from, prev CellID, buf []astar.Transition[CellID]) []astar.Transition[CellID] {
		buf = buf[:0]
		for _, n := range grid.Neighbors(from) {
			if n == prev {
				continue
			}
			cost, passable := terrain.MovementCost(n)
			if !passable || !occupancy.CanEnter(n, entity) {
				continue
			}
			buf = append(buf, astar.Transition[CellID]{To: n, Cost: cost})
		}
		return buf
	}
}

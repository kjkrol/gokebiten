package navigation

import (
	"github.com/kjkrol/astar"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/uid"
)

// pathFinder computes routes over one grid, reusing its A* solver across
// calls — build once and share across systems.
type pathFinder struct {
	grid      board.Grid
	terrain   board.Terrain
	occupancy board.Occupancy
	solver    *astar.Solver[board.CellID]
}

// newPathFinder builds a pathFinder over grid, respecting terrain/occupancy, allocating its solver once for reuse across findPath calls.
func newPathFinder(grid board.Grid, terrain board.Terrain, occupancy board.Occupancy) *pathFinder {
	return &pathFinder{
		grid: grid, terrain: terrain, occupancy: occupancy,
		solver: astar.New[board.CellID](func(a, b board.CellID) float64 { return grid.Distance(a, b) }),
	}
}

// findPath computes a route from 'from' toward 'to' for entity — ok=false if unreachable.
func (p *pathFinder) findPath(entity uid.UID64, from, to board.CellID) (Path, bool) {
	full := p.solver.Solve(from, to, p.transitionsFor(entity))
	if len(full) < 2 {
		return Path{}, false
	}
	steps := full[1:]
	n := min(len(steps), MaxPathLength)
	var path Path
	copy(path.Steps[:], steps[:n])
	path.Length = uint16(n)
	return path, true
}

// transitionsFor adapts p's Grid+Terrain+Occupancy into astar's Transitions shape for one entity's Solve call.
func (p *pathFinder) transitionsFor(entity uid.UID64) astar.Transitions[board.CellID] {
	return func(from, prev board.CellID, buf []astar.Transition[board.CellID]) []astar.Transition[board.CellID] {
		buf = buf[:0]
		for _, n := range p.grid.Neighbors(from) {
			if n == prev {
				continue
			}
			kind := p.terrain.Kind(n)
			if !kind.Passable || !p.occupancy.CanEnter(n, entity) {
				continue
			}
			if c1, c2, ok := p.grid.DiagonalNeighbors(from, n); ok {
				if !p.enterable(c1, entity) || !p.enterable(c2, entity) {
					continue
				}
			}
			buf = append(buf, astar.Transition[board.CellID]{To: n, Cost: kind.Cost * p.grid.NeighborCost(from, n)})
		}
		return buf
	}
}

// enterable reports whether entity may hold c: passable terrain nobody else occupies — the same test a diagonal step's corners must pass to be reserved.
func (p *pathFinder) enterable(c board.CellID, entity uid.UID64) bool {
	return p.terrain.Kind(c).Passable && p.occupancy.CanEnter(c, entity)
}

// maxVisitedCells bounds how many cells nearestFree inspects around its target.
const maxVisitedCells = 64

// nearestFree returns the cell nearest target that entity can hold and reach
// from from, skipping taken, together with the route to it.
func (p *pathFinder) nearestFree(entity uid.UID64, from, target board.CellID, taken map[board.CellID]bool) (board.CellID, Path, bool) {
	var path Path
	passable := func(c board.CellID) bool { return p.terrain.Kind(c).Passable }
	reachableFree := func(c board.CellID) bool {
		if taken[c] || !p.enterable(c, entity) {
			return false
		}
		if c == from {
			path = Path{}
			return true
		}
		found, ok := p.findPath(entity, from, c)
		path = found
		return ok
	}
	dest, ok := breadthFirst(target, p.grid.Neighbors, passable, reachableFree, maxVisitedCells)
	return dest, path, ok
}

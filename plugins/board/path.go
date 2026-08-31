package board

import (
	"github.com/kjkrol/astar"
	"github.com/kjkrol/uid"
)

// MaxPathLength bounds Path.Steps — a route longer than this is fetched in successive chunks.
const MaxPathLength = 64

// Path is the cached route toward MoveTo.Target, consumed step by step by
// SteeringSystem — Length 0 means "not computed yet".
type Path struct {
	Steps  [MaxPathLength]CellID
	Length uint16
	Index  uint16
}

// FindPath computes a route from 'from' toward 'to' for entity, respecting grid/terrain/occupancy — ok=false if unreachable.
func FindPath(grid Grid, terrain Terrain, occupancy Occupancy, entity uid.UID64, from, to CellID) (Path, bool) {
	solver := astar.New[CellID](func(a, b CellID) float64 { return grid.Distance(a, b) })
	return findPath(solver, grid, terrain, occupancy, entity, from, to)
}

func findPath(solver *astar.Solver[CellID], grid Grid, terrain Terrain, occupancy Occupancy, entity uid.UID64, from, to CellID) (Path, bool) {
	full := solver.Solve(from, to, transitionsFor(grid, terrain, occupancy, entity))
	if len(full) < 2 || full[len(full)-1] != to {
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

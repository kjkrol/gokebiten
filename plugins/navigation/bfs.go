package navigation

import "github.com/kjkrol/gram/plugins/board"

// breadthFirst returns the first cell match accepts, ring by ring from start through expand.
func breadthFirst(start board.CellID, neighbors func(board.CellID) []board.CellID,
	expand, match func(board.CellID) bool, maxVisited int) (board.CellID, bool) {
	queue := []board.CellID{start}
	visited := map[board.CellID]bool{start: true}
	for len(queue) > 0 && maxVisited > 0 {
		c := queue[0]
		queue = queue[1:]
		maxVisited--
		if match(c) {
			return c, true
		}
		for _, n := range neighbors(c) {
			if !visited[n] && expand(n) {
				visited[n] = true
				queue = append(queue, n)
			}
		}
	}
	return 0, false
}

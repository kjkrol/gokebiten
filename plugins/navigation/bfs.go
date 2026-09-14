package navigation

import "github.com/kjkrol/gokebiten/plugins/board"

// breadthFirst visits cells ring by ring outward from start — only through
// cells expand accepts, at most maxVisited — returning the first one match accepts.
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

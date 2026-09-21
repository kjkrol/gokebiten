package world

import "github.com/kjkrol/goke/v3"

// Modifier chains a per-entity transform of an accumulator V, in registration order.
// Bind adds what it reads to the host's query; the entity's Base is handed to Apply.
type Modifier[V any] interface {
	Bind(qb *goke.QueryBuilder)
	Apply(cur *goke.Cursor, i int, base *Base, acc V) V
}

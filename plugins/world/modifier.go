package world

import "github.com/kjkrol/goke/v3"

// Modifier chains a per-entity transform of an accumulator of type V — each
// bound Modifier sees what earlier ones left in acc, in registration order.
type Modifier[V any] interface {
	Bind(qb *goke.QueryBuilder)
	Apply(cur *goke.Cursor, i int, acc V) V
}

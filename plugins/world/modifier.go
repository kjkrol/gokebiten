package world

import "github.com/kjkrol/goke/v3"

// Modifier chains a per-entity transform of an accumulator of type V — each
// bound Modifier sees what earlier ones left in acc, in registration order.
//
// Bind adds whatever else the modifier reads to the host's query. The entity's
// Base is not among it: the host already requires Base, a query cannot carry a
// component twice, so Apply is handed it instead.
type Modifier[V any] interface {
	Bind(qb *goke.QueryBuilder)
	Apply(cur *goke.Cursor, i int, base *Base, acc V) V
}

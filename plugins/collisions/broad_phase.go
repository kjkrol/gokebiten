package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*BroadPhase)(nil)

type BroadPhase struct {
	space         *gokg.Space
	query         *goke.Query
	pos           goke.Comp[world.Position]
	vel           goke.Comp[world.Velocity]
	hit           goke.Comp[Hit]
	collision     goke.Comp[Collision]
	addEditor     *goke.Editor
	collidableIDs map[uid.UID64]struct{}
}

func NewBroadPhase(space *gokg.Space) *BroadPhase {
	return &BroadPhase{space: space, collidableIDs: make(map[uid.UID64]struct{})}
}

func (b *BroadPhase) Init(si *goke.SysInit) {
	b.query = si.NewQueryBuilder(&b.pos, &b.vel, &b.collision).Build()
	si.RegComp[Hit]()
	b.addEditor = b.query.NewEditorBuilder(&b.hit).Build()

	collidableQry := si.NewQueryBuilder().Include(goke.Include[Collision]()).Build()
	collidableQry.All()
	for collidableQry.Next() {
		for _, id := range collidableQry.Cursor().IDs {
			b.collidableIDs[id] = struct{}{}
		}
	}
}

func (b *BroadPhase) Update(cb *goke.CmdBuf, _ time.Duration) {
	const probeExpandMargin = 32
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		posSlice := b.pos.Slice(cursor)
		collisionSlice := b.collision.Slice(cursor)
		buf := b.query.BeginMigrate(cb)
		for i, entityA := range cursor.IDs {
			found := false
			p := &posSlice[i]
			c := &collisionSlice[i]

			checkFunc := func(boxA geom.AABB[uint32]) {
				b.space.Query(boxA, func(entityB uid.UID64, _ plane.FragPosition) {
					if entityA == entityB {
						return
					}
					if _, ok := b.collidableIDs[entityB]; !ok {
						return
					}
					c.addTouching(entityB)
					if !found {
						buf.Add(entityA)
						found = true
					}
				})
			}

			probeBoxA := p.AABB
			b.space.ExpandOnly(&probeBoxA, probeExpandMargin)
			checkFunc(probeBoxA.AABB)
			probeBoxA.VisitFragments(func(_ plane.FragPosition, boxA geom.AABB[uint32]) bool {
				checkFunc(boxA)
				return true
			})
		}
		buf.Commit(b.addEditor)
	}
}

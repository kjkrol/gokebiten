package collision

import (
	"github.com/kjkrol/aabbworld/collide"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*BroadPhase)(nil)

// BroadPhase names every two collidable entities close enough to touch this tick,
// and drops what each entity struck the tick before.
type BroadPhase struct {
	space    *aabbworld.Space
	found    *Candidates
	query    *goke.Query
	base     goke.Comp[world.Base]
	collider goke.Comp[Collider]

	// host runs the Each behaviors registered with the plugin, inside this pass.
	host *plugin.EachHost[Struck]

	// walking is the chunk the host is being run over.
	walking struct {
		ids       []uid.UID64
		colliders []Collider
	}
	struckAt func(i int) Struck
	onPair   func(a, b uid.UID64)
}

// NewBroadPhase builds the broad phase over space, recording its pairs in found.
func NewBroadPhase(space *aabbworld.Space, found *Candidates) *BroadPhase {
	return newBroadPhase(space, found, &plugin.EachHost[Struck]{})
}

func newBroadPhase(space *aabbworld.Space, found *Candidates, host *plugin.EachHost[Struck]) *BroadPhase {
	b := &BroadPhase{space: space, found: found, host: host}
	b.struckAt = b.struck
	b.onPair = found.Add
	return b
}

func (b *BroadPhase) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&b.base, &b.collider)
	b.host.Bind(qb)
	b.query = qb.Build()
}

func (b *BroadPhase) Update(cb *goke.CmdBuf, d time.Duration) {
	t := plugin.Tick{Cmd: cb, Now: time.Now(), Dt: d}
	marked := false
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		colliders := b.collider.Slice(cursor)

		b.walking.ids, b.walking.colliders = cursor.IDs, colliders
		b.host.Run(t, cursor, b.struckAt)
		for i, id := range cursor.IDs {
			c := &colliders[i]
			c.clearContacts()
			if !c.Indexed {
				b.space.SetCapabilities(id, aabbworld.CanCollide)
				c.Indexed, marked = true, true
			}
		}
	}
	if marked {
		b.space.Flush(nil)
	}

	b.found.reset()
	collide.BroadPhase(b.space, world.StepReach, aabbworld.CanCollide, b.onPair)
}

// struck is what the hosted behaviors are told about the i-th entity of the chunk being walked.
func (b *BroadPhase) struck(i int) Struck {
	return Struck{ID: b.walking.ids[i], Contacts: b.walking.colliders[i].Contacts()}
}

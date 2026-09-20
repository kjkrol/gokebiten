package collisions

import (
	"errors"
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokg"
)

var _ goke.Module = (*module)(nil)

// module is the optional broad/narrow-phase collision engine, run over a
// *gokg.Space borrowed from world.Module — see Plugin.Install.
type module struct {
	space  *gokg.Space
	margin float64
	ecs    *goke.ECS

	pairs    *plugin.PairHost[Meeting]
	entities *plugin.EachHost[Struck]

	broadPhase  goke.Runnable
	narrowPhase goke.Runnable
	built       bool
}

// New builds the collision engine over space. margin is how far the broad
// phase reaches past each entity when probing — see NewBroadPhase. Safe to
// call before ECS.Load, since systems register via RegSystems (see
// [goke.Module]).
func New(space *gokg.Space, ecs *goke.ECS, margin float64) *module {
	return newModule(space, ecs, margin, &plugin.PairHost[Meeting]{}, &plugin.EachHost[Struck]{})
}

func newModule(space *gokg.Space, ecs *goke.ECS, margin float64, pairs *plugin.PairHost[Meeting], entities *plugin.EachHost[Struck]) *module {
	return &module{space: space, margin: margin, ecs: ecs, pairs: pairs, entities: entities}
}

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems builds and registers the collision systems — see [goke.Module].
func (m *module) RegSystems(ecs *goke.ECS) {
	if !m.built {
		m.build()
	}
}

func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.broadPhase, d)
	ctx.Sync()

	ctx.Run(m.narrowPhase, d)
	ctx.Sync()
}

// SetupSystems is empty — the collision engine has no one-time seeding of its own.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types the collision engine owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Collision](),
		goke.LoadComp[Physics](),
	}
}

// =================================================================
// plugin.PostLoader contract
// =================================================================

// PostLoad gives every loaded entity carrying Collision the CanCollide
// capability — see plugin.PostLoader.
//
// This is the other half of [Collidable]: templates run only when entities are
// created, and a world restored from a save creates none.
func (m *module) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query := si.NewQueryBuilder().Include(goke.Include[Collision]()).Build()
		query.All()
		for query.Next() {
			for _, id := range query.Cursor().IDs {
				m.space.SetCapabilities(id, CanCollide)
			}
		}
		m.space.Flush(nil)
	}}
}

// =================================================================
// collisions-specific
// =================================================================

// RegisterBehavior hosts a plugin.Between behavior made for Meeting in the narrow
// phase, or a plugin.Each one made for Struck in the broad phase.
func (m *module) RegisterBehavior(behaviors ...plugin.Behavior) error {
	return host(m.pairs, m.entities, behaviors)
}

// host hands each behavior to whichever of the two hosts takes it, stopping at the first neither does.
func host(pairs *plugin.PairHost[Meeting], entities *plugin.EachHost[Struck], behaviors []plugin.Behavior) error {
	for _, b := range behaviors {
		err := pairs.Add(b)
		if errors.Is(err, plugin.ErrUnhostedBehavior) {
			err = entities.Add(b)
		}
		if errors.Is(err, plugin.ErrUnhostedBehavior) {
			return fmt.Errorf("%w in collisions — it takes Between for Meeting and Each for Struck", err)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *module) build() {
	m.broadPhase = m.ecs.RegSys(newBroadPhase(m.space, m.margin, m.entities))
	m.narrowPhase = m.ecs.RegSys(newNarrowPhase(m.space, m.pairs))
	m.built = true
}

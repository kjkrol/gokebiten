package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
)

var _ goke.Module = (*module)(nil)

// module is the optional broad/narrow-phase collision engine, run over a
// *gokg.Space borrowed from world.Module — see Plugin.Install.
type module struct {
	space  *gokg.Space
	margin float64
	ecs    *goke.ECS

	broadPhase  goke.Runnable
	narrowPhase goke.Runnable
	built       bool
}

// New builds the collision engine over space. margin is how far the broad
// phase reaches past each entity when probing — see NewBroadPhase. Safe to
// call before ECS.Load, since systems register via RegSystems (see
// [goke.Module]).
func New(space *gokg.Space, ecs *goke.ECS, margin float64) *module {
	return &module{space: space, margin: margin, ecs: ecs}
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
		goke.LoadComp[Contacts](),
		goke.LoadComp[Mass](),
		goke.LoadComp[Restitution](),
		goke.LoadComp[Sensor](),
		goke.LoadComp[Static](),
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

func (m *module) build() {
	m.broadPhase = m.ecs.RegSys(NewBroadPhase(m.space, m.margin))
	m.narrowPhase = m.ecs.RegSys(NewNarrowPhase(m.space))
	m.built = true
}

package collision

import (
	"errors"
	"fmt"
	"github.com/kjkrol/uid"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
)

var _ goke.Module = (*module)(nil)

// module is the optional collision engine, run over a *aabbworld.Space borrowed from world.
type module struct {
	space *aabbworld.Space
	ecs   *goke.ECS

	pairs    *plugin.PairHost[Meeting]
	entities *plugin.EachHost[Struck]

	detector goke.Runnable
	tracked  func(t plugin.Tick, id uid.UID64, inside bool)
	shapes   ShapeTest
	built    bool
}

// New builds the collision engine over space.
func New(space *aabbworld.Space, ecs *goke.ECS) *module {
	return newModule(space, ecs, &plugin.PairHost[Meeting]{}, &plugin.EachHost[Struck]{})
}

func newModule(space *aabbworld.Space, ecs *goke.ECS, pairs *plugin.PairHost[Meeting], entities *plugin.EachHost[Struck]) *module {
	return &module{space: space, ecs: ecs, pairs: pairs, entities: entities}
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
	ctx.Run(m.detector, d)
	ctx.Sync()
}

// SetupSystems is empty — the collision engine has no one-time seeding of its own.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types the collision engine owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Collider](),
		goke.LoadComp[Physics](),
	}
}

// =================================================================
// collision-specific
// =================================================================

// RegisterBehavior hosts a plugin.Between of Meeting or a plugin.Each of Struck.
func (m *module) RegisterBehavior(behaviors ...plugin.Behavior) error {
	return host(m.pairs, m.entities, behaviors)
}

// host hands each behavior to whichever host takes it, stopping at the first neither does.
func host(pairs *plugin.PairHost[Meeting], entities *plugin.EachHost[Struck], behaviors []plugin.Behavior) error {
	for _, b := range behaviors {
		err := pairs.Add(b)
		if errors.Is(err, plugin.ErrUnhostedBehavior) {
			err = entities.Add(b)
		}
		if errors.Is(err, plugin.ErrUnhostedBehavior) {
			return fmt.Errorf("%w in collision — it takes Between for Meeting and Each for Struck", err)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *module) build() {
	detector := newDetector(m.space, m.pairs, m.entities, m.shapes)
	if m.tracked != nil {
		detector.tracked = m.tracked
	}
	m.detector = m.ecs.RegSys(detector)
	m.built = true
}

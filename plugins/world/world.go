package world

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/uid"
)

// Config configures a world.World: its spatial shape and the bounds its entity population must respect.
type Config struct {
	Space    SpaceCfg
	Entities EntitiesCfg
}

type SpaceCfg struct {
	Width, Height uint32
	Toroidal      bool
}

type EntitiesCfg struct {
	MaxCount int
	MinSize  uint32
	MaxSize  uint32
}

// Telemetry is how many entities Populate/PostLoad have created — see World.Telemetry.
type Telemetry struct {
	Count int
}

// entityExtras supplies the components and per-entity values a spawn needs
// beyond Position/Velocity — newComponentAdder is the only implementation;
// attach WithEffect for a side effect (like updating an occupancy tracker)
// that must run alongside the write.
type entityExtras interface {
	Components() []goke.Addable
	Init(cursor *goke.Cursor, i, index int, id uid.UID64)
}

// SpeedModifier contributes a multiplicative factor to an entity's Velocity.Value each tick — see VelocitySystem.
type SpeedModifier = Modifier[float64]

// World owns the world's topology, entities, and movement — the mandatory
// foundation any game with moving, drawable entities builds on.
type World struct {
	config Config
	space  *gokg.Space
	step   time.Duration
	ecs    *goke.ECS

	spawnedCount int
	seeds        []goke.System
	telemetry    Telemetry

	modifiers        []SpeedModifier
	velocityRunnable goke.Runnable
	moveRunnable     goke.Runnable
	built            bool
}

var _ goke.SetupProvider = (*World)(nil)
var _ goke.CompProvider = (*World)(nil)

// NewWorld builds the world's topology and spatial index from cfg.
func NewWorld(cfg Config) *World {
	return &World{config: cfg, space: buildSpace(cfg)}
}

// Config returns the Config this World was built from.
func (w *World) Config() Config { return w.config }

// Telemetry returns the entity-count telemetry Populate/PostLoad maintain.
func (w *World) Telemetry() *Telemetry { return &w.telemetry }

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (w *World) RegisterSpeedModifier(m SpeedModifier) { w.modifiers = append(w.modifiers, m) }

// SetupSystems runs every queued Populate call, in call order.
func (w *World) SetupSystems() []goke.System { return w.seeds }

// RunPlan runs world's movement pipeline (speed modifiers, then integration) for this tick.
func (w *World) RunPlan(ctx goke.RunCtx, d time.Duration) {
	if !w.built {
		w.build()
	}
	ctx.Run(w.velocityRunnable, d)
	ctx.Run(w.moveRunnable, d)
	ctx.Sync()
}

func (w *World) build() {
	velocitySystem := NewVelocitySystem(w.modifiers, w.maxSpeed())
	moveSystem := NewMoveSystem(w.space)
	w.velocityRunnable = w.ecs.RegSys(velocitySystem)
	w.moveRunnable = w.ecs.RegSys(moveSystem)
	w.built = true
}

// LoadComps lists the component types world owns — see [goke.CompProvider].
func (w *World) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Position](),
		goke.LoadComp[Appearance](),
		goke.LoadComp[Velocity](),
	}
}

// PostLoad recomputes Count and reinserts every loaded entity's Position into space — see gokebiten.PostLoader.
func (w *World) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[Position]
		query := si.NewQueryBuilder(&pos).Build()
		query.All()
		count := 0
		for query.Next() {
			cursor := query.Cursor()
			positions := pos.Slice(cursor)
			for i, id := range cursor.IDs {
				w.space.Insert(id, positions[i].AABB)
			}
			count += len(cursor.IDs)
		}
		w.space.Flush(nil)
		w.telemetry.Count = count
	}}
}

// maxSpeed caps displacement per tick to half the smallest entity's size, so
// nothing can tunnel through another entity undetected — 0 (no cap) if
// Config.Entities.MinSize/the tick step aren't set.
func (w *World) maxSpeed() int32 {
	if w.step <= 0 || w.config.Entities.MinSize == 0 {
		return 0
	}
	return int32(float64(w.config.Entities.MinSize) / 2 / w.step.Seconds())
}

func (w *World) reserve(count int) {
	if w.spawnedCount+count > w.config.Entities.MaxCount {
		panic(fmt.Sprintf("world: spawning %d more would exceed Config.Entities.MaxCount %d (already spawned %d)",
			count, w.config.Entities.MaxCount, w.spawnedCount))
	}
	w.spawnedCount += count
}

func (w *World) validateSize(id uid.UID64, pos Position) {
	if pos.Size.X < w.config.Entities.MinSize || pos.Size.X > w.config.Entities.MaxSize ||
		pos.Size.Y < w.config.Entities.MinSize || pos.Size.Y > w.config.Entities.MaxSize {
		panic(fmt.Sprintf("world: entity %d size %dx%d outside declared bounds [%d, %d]",
			id, pos.Size.X, pos.Size.Y, w.config.Entities.MinSize, w.config.Entities.MaxSize))
	}
}

// Populate queues a spawn of count entities — see ExampleWorld_Populate.
func (w *World) Populate(count int, spawner *Spawner) *World {
	w.seeds = append(w.seeds, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		w.populateDynamic(si, spawner, count)
	}})
	return w
}

func (w *World) populateDynamic(si *goke.SysInit, spawner *Spawner, count int) {
	w.reserve(count)
	w.telemetry.Count += count

	var posComp goke.Comp[Position]
	var velComp goke.Comp[Velocity]
	comps := []goke.Addable{&posComp, &velComp}
	for _, e := range spawner.extras {
		comps = append(comps, e.Components()...)
	}
	factory := si.NewFactory(comps...)

	factory.Create(count)
	index := 0
	for factory.Next() {
		positions := posComp.Slice(&factory.Cursor)
		velocities := velComp.Slice(&factory.Cursor)
		for i, id := range factory.IDs {
			pos := spawner.position(index, count)
			vel := spawner.velocity(index)
			w.validateSize(id, pos)
			positions[i] = pos
			velocities[i] = vel
			w.space.Insert(id, pos.AABB)
			for _, e := range spawner.extras {
				e.Init(&factory.Cursor, i, index, id)
			}
			index++
		}
	}
	w.space.Flush(nil)
}

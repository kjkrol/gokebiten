package world

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/uid"
)

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

// module owns the world's topology, entities, and movement — the mandatory
// foundation any game with moving, drawable entities builds on.
type module struct {
	config Config
	space  *gokg.Space
	step   time.Duration

	spawnedCount int
	seeds        []goke.System
	telemetry    Telemetry

	modifiers        []SpeedModifier
	velocityRunnable goke.Runnable
	moveRunnable     goke.Runnable
}

var _ goke.Module = (*module)(nil)

// newModule builds the world's topology and spatial index from cfg.
func newModule(cfg Config) *module {
	return &module{config: cfg, space: buildSpace(cfg)}
}

func (w *module) Config() Config { return w.config }

func (w *module) Telemetry() *Telemetry { return &w.telemetry }

func (w *module) RegisterSpeedModifier(m SpeedModifier) { w.modifiers = append(w.modifiers, m) }

// SetupSystems runs every queued Populate call, in call order.
func (w *module) SetupSystems() []goke.System { return w.seeds }

// RunPlan runs world's movement pipeline (speed modifiers, then integration) for this tick.
func (w *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(w.velocityRunnable, d)
	ctx.Run(w.moveRunnable, d)
	ctx.Sync()
}

// RegSystems registers world's movement systems — see [goke.Module].
func (w *module) RegSystems(ecs *goke.ECS) {
	if w.velocityRunnable != nil {
		return
	}
	velocitySystem := NewVelocitySystem(w.modifiers, w.maxSpeed())
	moveSystem := NewMoveSystem(w.space)
	w.velocityRunnable = ecs.RegSys(velocitySystem)
	w.moveRunnable = ecs.RegSys(moveSystem)
}

// LoadComps lists the component types world owns — see [goke.CompProvider].
func (w *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Position](),
		goke.LoadComp[Appearance](),
		goke.LoadComp[Velocity](),
	}
}

// PostLoad recomputes Count and reinserts every loaded entity's Position into space — see gokebiten.PostLoader.
func (w *module) PostLoad() goke.System {
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
func (w *module) maxSpeed() int32 {
	if w.step <= 0 || w.config.Entities.MinSize == 0 {
		return 0
	}
	return int32(float64(w.config.Entities.MinSize) / 2 / w.step.Seconds())
}

func (w *module) reserve(count int) {
	if w.spawnedCount+count > w.config.Entities.MaxCount {
		panic(fmt.Sprintf("world: spawning %d more would exceed Config.Entities.MaxCount %d (already spawned %d)",
			count, w.config.Entities.MaxCount, w.spawnedCount))
	}
	w.spawnedCount += count
}

func (w *module) validateSize(id uid.UID64, pos Position) {
	if pos.Size.X < w.config.Entities.MinSize || pos.Size.X > w.config.Entities.MaxSize ||
		pos.Size.Y < w.config.Entities.MinSize || pos.Size.Y > w.config.Entities.MaxSize {
		panic(fmt.Sprintf("world: entity %d size %dx%d outside declared bounds [%d, %d]",
			id, pos.Size.X, pos.Size.Y, w.config.Entities.MinSize, w.config.Entities.MaxSize))
	}
}

// Populate queues a spawn of count entities — see ExamplePlugin_Populate.
func (w *module) Populate(count int, spawner *Spawner) *module {
	w.seeds = append(w.seeds, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		w.populateDynamic(si, spawner, count)
	}})
	return w
}

func (w *module) populateDynamic(si *goke.SysInit, spawner *Spawner, count int) {
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

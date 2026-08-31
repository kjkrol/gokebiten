package world

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/uid"
)

type Config struct {
	Width, Height uint32
	Toroidal      bool
}

type Population struct {
	MaxCount int
	MinSize  uint32
	MaxSize  uint32
}

// Telemetry is how many entities Populate/PostLoad have created — see Module.Telemetry.
type Telemetry struct {
	Count int
}

// EntityExtras supplies the components and per-entity values a spawn needs beyond Position/Velocity.
type EntityExtras interface {
	Components() []goke.Addable
	Init(cursor *goke.Cursor, i, index int, id uid.UID64)
}

// SpeedModifier contributes a multiplicative factor to an entity's Velocity.Value each tick — see VelocitySystem.
type SpeedModifier = Modifier[float64]

// Module owns the world's topology, entities, and movement — the mandatory
// foundation any game with moving, drawable entities builds on.
type Module struct {
	config     Config
	population Population
	space      *gokg.Space
	step       time.Duration
	ecs        *goke.ECS

	spawnedCount int
	seeds        []goke.System
	telemetry    Telemetry

	modifiers        []SpeedModifier
	velocityRunnable goke.Runnable
	moveRunnable     goke.Runnable
	built            bool
}

var _ goke.SetupProvider = (*Module)(nil)
var _ goke.CompProvider = (*Module)(nil)

// NewModule builds the world's topology and spatial index from cfg, provisioned for pop.
func NewModule(cfg Config, pop Population) *Module {
	return &Module{config: cfg, population: pop, space: buildSpace(cfg, pop)}
}

func (w *Module) Population() Population { return w.population }

// Telemetry returns the entity-count telemetry Populate/PostLoad maintain.
func (w *Module) Telemetry() *Telemetry { return &w.telemetry }

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (w *Module) RegisterSpeedModifier(m SpeedModifier) { w.modifiers = append(w.modifiers, m) }

// SetupSystems runs every queued Populate call, in call order.
func (w *Module) SetupSystems() []goke.System { return w.seeds }

// RunPlan runs world's movement pipeline (speed modifiers, then integration) for this tick.
func (w *Module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	if !w.built {
		w.build()
	}
	ctx.Run(w.velocityRunnable, d)
	ctx.Run(w.moveRunnable, d)
	ctx.Sync()
}

func (w *Module) build() {
	velocitySystem := NewVelocitySystem(w.modifiers, w.maxSpeed())
	moveSystem := NewMoveSystem(w.space)
	w.velocityRunnable = w.ecs.RegSys(velocitySystem)
	w.moveRunnable = w.ecs.RegSys(moveSystem)
	w.built = true
}

// LoadComps lists the component types world owns — see [goke.CompProvider].
func (w *Module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Position](),
		goke.LoadComp[Appearance](),
		goke.LoadComp[Velocity](),
	}
}

// PostLoad recomputes Count and reinserts every loaded entity's Position into space — see gokebiten.PostLoader.
func (w *Module) PostLoad() goke.System {
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

// maxSpeed caps displacement per tick to half the smallest population entity's
// size, so nothing can tunnel through another entity undetected — 0 (no cap)
// if Population.MinSize/the tick step aren't set.
func (w *Module) maxSpeed() int32 {
	if w.step <= 0 || w.population.MinSize == 0 {
		return 0
	}
	return int32(float64(w.population.MinSize) / 2 / w.step.Seconds())
}

func (w *Module) reserve(count int) {
	if w.spawnedCount+count > w.population.MaxCount {
		panic(fmt.Sprintf("world: spawning %d more would exceed Population.MaxCount %d (already spawned %d)",
			count, w.population.MaxCount, w.spawnedCount))
	}
	w.spawnedCount += count
}

func (w *Module) validateSize(id uid.UID64, pos Position) {
	if pos.Size.X < w.population.MinSize || pos.Size.X > w.population.MaxSize ||
		pos.Size.Y < w.population.MinSize || pos.Size.Y > w.population.MaxSize {
		panic(fmt.Sprintf("world: entity %d size %dx%d outside declared population bounds [%d, %d]",
			id, pos.Size.X, pos.Size.Y, w.population.MinSize, w.population.MaxSize))
	}
}

// Populate queues a spawn of count entities from populators — populators[0] must be a Spawner or a Placement.
func (w *Module) Populate(count int, populators ...EntityExtras) *Module {
	w.seeds = append(w.seeds, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		w.populate(si, count, populators...)
	}})
	return w
}

func (w *Module) populate(si *goke.SysInit, count int, populators ...EntityExtras) {
	if len(populators) == 0 {
		panic("world: Populate requires a world.Spawner or world.Placement as its first populator")
	}
	switch first := populators[0].(type) {
	case Spawner:
		w.populateDynamic(si, first, count, populators...)
	case Placement:
		w.populateStatic(si, first, count, populators...)
	default:
		panic("world: Populate's first populator must implement world.Spawner or world.Placement")
	}
}

func (w *Module) populateDynamic(si *goke.SysInit, spawner Spawner, count int, populators ...EntityExtras) {
	w.reserve(count)
	w.telemetry.Count += count

	var posComp goke.Comp[Position]
	var velComp goke.Comp[Velocity]
	comps := []goke.Addable{&posComp, &velComp}
	for _, p := range populators {
		comps = append(comps, p.Components()...)
	}
	factory := si.NewFactory(comps...)

	factory.Create(count)
	index := 0
	for factory.Next() {
		positions := posComp.Slice(&factory.Cursor)
		velocities := velComp.Slice(&factory.Cursor)
		for i, id := range factory.IDs {
			pos, vel := spawner.Spawn(index, count)
			w.validateSize(id, pos)
			positions[i] = pos
			velocities[i] = vel
			w.space.Insert(id, pos.AABB)
			for _, p := range populators {
				p.Init(&factory.Cursor, i, index, id)
			}
			index++
		}
	}
	w.space.Flush(nil)
}

func (w *Module) populateStatic(si *goke.SysInit, placement Placement, count int, populators ...EntityExtras) {
	w.reserve(count)
	w.telemetry.Count += count

	var posComp goke.Comp[Position]
	comps := []goke.Addable{&posComp}
	for _, p := range populators {
		comps = append(comps, p.Components()...)
	}
	factory := si.NewFactory(comps...)

	factory.Create(count)
	index := 0
	for factory.Next() {
		positions := posComp.Slice(&factory.Cursor)
		for i, id := range factory.IDs {
			pos := placement.Place(index, count)
			w.validateSize(id, pos)
			positions[i] = pos
			w.space.Insert(id, pos.AABB)
			for _, p := range populators {
				p.Init(&factory.Cursor, i, index, id)
			}
			index++
		}
	}
	w.space.Flush(nil)
}

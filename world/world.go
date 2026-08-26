package world

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/spatial"
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

// EntityExtras supplies the components and per-entity values a spawn needs beyond Position/Velocity.
type EntityExtras interface {
	Components() []goke.Addable
	Init(cursor *goke.Cursor, i, index int, id uid.UID64)
}

// Module owns the spatial index and population bookkeeping for one game.
type Module struct {
	space        *gokg.Space
	population   Population
	spawnedCount int
	seeds        []goke.System
	onReindexed  func(count int)
}

var _ goke.SetupProvider = (*Module)(nil)

// NewModule builds the spatial index from cfg, provisioned for pop.
func NewModule(cfg Config, pop Population) *Module {
	return &Module{space: buildSpace(cfg, pop), population: pop}
}

func (w *Module) Space() *gokg.Space     { return w.space }
func (w *Module) Population() Population { return w.population }

// SetupSystems runs every queued Populate/PopulateStatic call, in call order.
func (w *Module) SetupSystems() []goke.System { return w.seeds }

// PostLoad rebuilds the spatial index from every loaded entity's kinematics.Position — see gokebiten.PostLoader.
func (w *Module) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[kinematics.Position]
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
		if w.onReindexed != nil {
			w.onReindexed(count)
		}
	}}
}

// Populate queues a spawn of count moving entities from populators (populators[0] must be a kinematics.Spawner).
func (w *Module) Populate(count int, telemetry *kinematics.Telemetry, populators ...EntityExtras) *Module {
	w.seeds = append(w.seeds, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		w.populate(si, count, telemetry, populators...)
	}})
	return w
}

// PopulateStatic queues a spawn of count static entities from populators (populators[0] must be a kinematics.Placement).
func (w *Module) PopulateStatic(count int, telemetry *kinematics.Telemetry, populators ...EntityExtras) *Module {
	var posComp goke.Comp[kinematics.Position]
	var staticTag goke.Comp[collisions.Static]
	var q *goke.Query
	var staticEditor *goke.Editor
	var ids []uid.UID64

	w.seeds = append(w.seeds, goke.SystemFn{
		OnInit: func(si *goke.SysInit) {
			ids = w.populateStatic(si, &posComp, count, telemetry, populators...)
			q = si.NewQueryBuilder(&posComp).Build()
			staticEditor = q.NewEditorBuilder(&staticTag).Build()
		},
		OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
			q.Pick(ids)
			for q.Next() {
				buf := q.BeginMigrate(cb)
				for _, id := range q.Cursor().IDs {
					buf.Add(id)
				}
				buf.Commit(staticEditor)
			}
		},
	})
	return w
}

func buildSpace(cfg Config, pop Population) *gokg.Space {
	const minCapacity, maxCapacity = 2.0, 8.0

	worldArea := uint64(cfg.Width) * uint64(cfg.Height)
	entityArea := uint64(pop.MaxSize) * uint64(pop.MaxSize)
	density := float64(uint64(pop.MaxCount)*entityArea) / float64(worldArea)

	raw := math.Round(1.0 / math.Sqrt(density))
	capacity := uint32(math.Max(minCapacity, math.Min(maxCapacity, raw)))
	bucketResolution := spatial.ResolutionFrom(pop.MaxSize * capacity)

	log.Printf("[world] maxEntities=%d, density=%.2f%%, capacity=%d → bucket=%dx%d, bucketCap=%d, opsBuffer=%d",
		pop.MaxCount, density*100, capacity,
		bucketResolution.Side(), bucketResolution.Side(),
		capacity*capacity, pop.MaxCount*8)

	space, err := gokg.NewSpace(gokg.Config{
		Width:          cfg.Width,
		Height:         cfg.Height,
		Toroidal:       cfg.Toroidal,
		BucketSize:     bucketResolution,
		BucketCapacity: int(capacity * capacity),
		OpsBufferSize:  pop.MaxCount * 8,
	})
	if err != nil {
		panic(fmt.Sprintf("world: invalid space configuration: %v", err))
	}
	return space
}

func (w *Module) reserve(count int) {
	if w.spawnedCount+count > w.population.MaxCount {
		panic(fmt.Sprintf("world: spawning %d more would exceed Population.MaxCount %d (already spawned %d)",
			count, w.population.MaxCount, w.spawnedCount))
	}
	w.spawnedCount += count
}

func (w *Module) validateSize(id uid.UID64, pos kinematics.Position) {
	if pos.Size.X < w.population.MinSize || pos.Size.X > w.population.MaxSize ||
		pos.Size.Y < w.population.MinSize || pos.Size.Y > w.population.MaxSize {
		panic(fmt.Sprintf("world: entity %d size %dx%d outside declared population bounds [%d, %d]",
			id, pos.Size.X, pos.Size.Y, w.population.MinSize, w.population.MaxSize))
	}
}

func (w *Module) populate(si *goke.SysInit, count int, telemetry *kinematics.Telemetry, populators ...EntityExtras) {
	if len(populators) == 0 {
		panic("world: Populate requires a kinematics.Spawner as its first populator")
	}
	spawner, ok := populators[0].(kinematics.Spawner)
	if !ok {
		panic("world: Populate's first populator must implement kinematics.Spawner")
	}
	w.reserve(count)
	telemetry.DynamicCount = count

	var posComp goke.Comp[kinematics.Position]
	var velComp goke.Comp[kinematics.Velocity]
	var collisionComp goke.Comp[collisions.Collision]
	comps := []goke.Addable{&posComp, &velComp, &collisionComp}
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
			for _, p := range populators {
				p.Init(&factory.Cursor, i, index, id)
			}

			w.space.Insert(id, positions[i].AABB)
			index++
		}
	}
	w.space.Flush(nil)
}

func (w *Module) populateStatic(si *goke.SysInit, posComp *goke.Comp[kinematics.Position], count int, telemetry *kinematics.Telemetry, populators ...EntityExtras) []uid.UID64 {
	if len(populators) == 0 {
		panic("world: PopulateStatic requires a kinematics.Placement as its first populator")
	}
	placement, ok := populators[0].(kinematics.Placement)
	if !ok {
		panic("world: PopulateStatic's first populator must implement kinematics.Placement")
	}
	w.reserve(count)
	telemetry.StaticCount = count

	comps := []goke.Addable{posComp}
	for _, p := range populators {
		comps = append(comps, p.Components()...)
	}
	factory := si.NewFactory(comps...)

	ids := make([]uid.UID64, 0, count)
	factory.Create(count)
	index := 0
	for factory.Next() {
		positions := posComp.Slice(&factory.Cursor)
		for i, id := range factory.IDs {
			pos := placement.Place(index, count)
			w.validateSize(id, pos)
			positions[i] = pos
			for _, p := range populators {
				p.Init(&factory.Cursor, i, index, id)
			}
			ids = append(ids, id)

			w.space.Insert(id, positions[i].AABB)
			index++
		}
	}
	w.space.Flush(nil)
	return ids
}

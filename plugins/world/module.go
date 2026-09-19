package world

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/uid"
)

// SpeedModifier contributes a multiplicative factor to an entity's Velocity.Value each tick — see VelocitySystem.
type SpeedModifier = Modifier[float64]

// module owns the world's topology, entities, and movement — the foundation
// any Stage with moving, drawable entities builds on.
type module struct {
	config Config
	space  *gokg.Space

	spawnedCount int
	seeds        []goke.System
	telemetry    Telemetry

	// despawned is this tick's removals, so asking twice for the same entity
	// costs the world one slot, not two.
	despawned map[uid.UID64]struct{}

	entKinds *EntKindDict

	behaviors         []Behavior
	behaviorRunnables []goke.Runnable

	modifiers        []SpeedModifier
	steeringRunnable goke.Runnable
	velocityRunnable goke.Runnable
	moveRunnable     goke.Runnable
}

var _ goke.Module = (*module)(nil)

// newModule builds the world's topology and spatial index from cfg.
func newModule(cfg Config) *module {
	return &module{config: cfg, space: buildSpace(cfg), despawned: make(map[uid.UID64]struct{})}
}

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems registers world's movement systems — see [goke.Module].
func (w *module) RegSystems(ecs *goke.ECS) {
	if w.velocityRunnable != nil {
		return
	}
	for _, b := range w.behaviors {
		w.behaviorRunnables = append(w.behaviorRunnables, ecs.RegSys(b))
	}
	w.steeringRunnable = ecs.RegSys(NewSteeringSystem())
	velocitySystem := NewVelocitySystem(w.modifiers)
	moveSystem := NewMoveSystem(w.space, w.maxStep())
	w.velocityRunnable = ecs.RegSys(velocitySystem)
	w.moveRunnable = ecs.RegSys(moveSystem)
}

// maxStep is the furthest one entity may travel in a single tick. MoveSystem
// clamps to it so nothing tunnels through a neighbour; anything that has to
// notice an entity before it arrives — a broad phase, say — has to reach at
// least this far ahead, which is why it is named here rather than inlined.
func (w *module) maxStep() float64 { return float64(w.config.Entities.MinSize) / 2 }

// RunPlan runs world's tick: decisions first, then the movement pipeline
// (speed modifiers, then integration) that acts on them.
func (w *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	clear(w.despawned)
	for _, b := range w.behaviorRunnables {
		ctx.Run(b, d)
		ctx.Sync()
	}
	ctx.Run(w.steeringRunnable, d)
	ctx.Run(w.velocityRunnable, d)
	ctx.Run(w.moveRunnable, d)
	ctx.Sync()
}

// SetupSystems runs every queued Populate call, in call order.
func (w *module) SetupSystems() []goke.System { return w.seeds }

// LoadComps lists the component types world owns — see [goke.CompProvider].
func (w *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Position](),
		goke.LoadComp[Appearance](),
		goke.LoadComp[Velocity](),
		goke.LoadComp[Type](),
		goke.LoadComp[Steering](),
	}
}

// =================================================================
// plugin.PostLoader contract
// =================================================================

// PostLoad recomputes Count and reinserts every loaded entity's Position into space — see plugin.PostLoader.
func (w *module) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		w.remapTypes(si)

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

// remapTypes rewrites every loaded Type.ID through the saved dictionary, so a
// world keeps its kinds even when the game's Define order changed since the
// save. EntKindDict.order is this build's mapping and gob never touches it;
// EntKindDict.saved is what the save brought in.
func (w *module) remapTypes(si *goke.SysInit) {
	saved := w.entKinds.saved

	lut := make([]TypeID, len(saved))
	moved := false
	for old, name := range saved {
		k, ok := w.entKinds.Get(name)
		if !ok {
			panic(fmt.Sprintf("world: the save names EntKind %q, which this build no longer defines", name))
		}
		lut[old] = k.TypeID
		moved = moved || k.TypeID != TypeID(old)
	}
	if !moved {
		return
	}

	var typ goke.Comp[Type]
	query := si.NewQueryBuilder(&typ).Build()
	query.All()
	for query.Next() {
		cursor := query.Cursor()
		types := typ.Slice(cursor)
		for i := range cursor.IDs {
			if int(types[i].ID) >= len(lut) {
				panic(fmt.Sprintf("world: loaded entity carries TypeID %d, beyond the %d the save named", types[i].ID, len(lut)))
			}
			types[i].ID = lut[types[i].ID]
		}
	}
}

// =================================================================
// world-specific
// =================================================================

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (w *module) RegisterSpeedModifier(m SpeedModifier) { w.modifiers = append(w.modifiers, m) }

// RegisterBehavior adds b to the decision pass that runs before movement.
func (w *module) RegisterBehavior(b Behavior) { w.behaviors = append(w.behaviors, b) }

// despawn drops id from the ECS and from the spatial index, once per tick however often it is asked.
func (w *module) despawn(cb *goke.CmdBuf, id uid.UID64) {
	if _, gone := w.despawned[id]; gone {
		return
	}
	w.despawned[id] = struct{}{}
	cb.RemoveOne(id)
	w.space.Remove(id)
	w.spawnedCount--
	w.telemetry.Count--
}

// populate queues a spawn of one entity of kind per element of data, each element feeding kind's Load templates.
func (w *module) populate(kind EntKind, data []any) {
	count := len(data)
	extras := []entityExtras{
		Const(Appearance{SpriteID: kind.SpriteID}).adder(),
		Const(Type{ID: kind.TypeID}).adder(),
	}
	for _, c := range kind.Components {
		extras = append(extras, c.adder())
	}

	w.seeds = append(w.seeds, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		w.reserve(count)
		w.telemetry.Count += count

		var posComp goke.Comp[Position]
		var velComp goke.Comp[Velocity]
		comps := []goke.Addable{&posComp, &velComp}
		for _, e := range extras {
			comps = append(comps, e.Components()...)
		}
		factory := si.NewFactory(comps...)

		factory.Create(count)
		index := 0
		for factory.Next() {
			positions := posComp.Slice(&factory.Cursor)
			velocities := velComp.Slice(&factory.Cursor)
			for i, id := range factory.IDs {
				d := data[index]
				pos := kind.Position.resolve(d, id)
				w.validateSize(id, pos)
				positions[i] = pos
				velocities[i] = kind.Velocity.resolve(d, id)
				w.space.Insert(id, pos.AABB)
				for _, e := range extras {
					e.Init(&factory.Cursor, i, d, id)
				}
				index++
			}
		}
		w.space.Flush(nil)
	}})
}

func (w *module) validateSize(id uid.UID64, pos Position) {
	if pos.Size.X < float64(w.config.Entities.MinSize) || pos.Size.X > float64(w.config.Entities.MaxSize) ||
		pos.Size.Y < float64(w.config.Entities.MinSize) || pos.Size.Y > float64(w.config.Entities.MaxSize) {
		panic(fmt.Sprintf("world: entity %d size %vx%v outside declared bounds [%d, %d]",
			id, pos.Size.X, pos.Size.Y, w.config.Entities.MinSize, w.config.Entities.MaxSize))
	}
}

func (w *module) reserve(count int) {
	if w.spawnedCount+count > w.config.Entities.MaxCount {
		panic(fmt.Sprintf("world: spawning %d more would exceed Config.Entities.MaxCount %d (already spawned %d)",
			count, w.config.Entities.MaxCount, w.spawnedCount))
	}
	w.spawnedCount += count
}

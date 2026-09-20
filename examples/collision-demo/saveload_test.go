package main

import (
	"fmt"
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/hit"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

// testInstallCtx is a minimal plugin.Installer for tests that call Install directly.
type testInstallCtx struct {
	ecs     *goke.ECS
	pending []func() []goke.System
	tracked []any
}

func (c *testInstallCtx) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(si *goke.SysInit) { m.RegSystems(c.ecs) }}
	c.tracked = append(c.tracked, m)
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}
func (c *testInstallCtx) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.tracked = append(c.tracked, p)
		c.pending = append(c.pending, p.SetupSystems)
	}
}
func (c *testInstallCtx) RegSys(factory func() goke.System) goke.Runnable {
	return c.ecs.RegSys(factory())
}
func (c *testInstallCtx) ECS() *goke.ECS { return c.ecs }

// eachOnce drops repeated tokens, as the engine does: a kind and a module may
// both name a type — Collision here — and Load refuses to be told twice.
func eachOnce(tokens []goke.CompToken) []goke.CompToken {
	listed := map[string]bool{}
	var once []goke.CompToken
	for _, token := range tokens {
		if !listed[token.Name] {
			listed[token.Name] = true
			once = append(once, token)
		}
	}
	return once
}

// countCollidable reports how many entities the space will offer as collision
// candidates — the only way to observe, from outside, that something actually
// carries CanCollide.
func countCollidable(space *gokg.Space) int {
	box := plane.NewAABB(geom.NewVec(0, 0), ScreenWidth-1, ScreenHeight-1)
	seen := map[uid.UID64]struct{}{}
	space.Neighbours(&box, 0, collisions.CanCollide, func(id uid.UID64, _ plane.FragPosition) {
		seen[id] = struct{}{}
	})
	return len(seen)
}

// TestSaveLoadCycle exercises the same mechanics Persistence.Save/Load use, below the level of Engine (no Ebiten window).
func TestSaveLoadCycle(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	const count = 5
	cfg := world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: count, MinSize: RectSize, MaxSize: RectSize},
	}

	ecs := goke.New()
	wp := world.NewPlugin(cfg)
	placement := world.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize)
	motion := newRandomVelocity(200, 50, 10)
	// Both halves define the kinds, as a Stage's Init does before it ever loads:
	// the dictionary is what tells Load about hit.Mark, which no module owns.
	defineKinds := func(wp *world.Plugin) []world.Entry {
		kinds := wp.EntKindDict()
		var entries []world.Entry
		for i := range count {
			name := fmt.Sprintf("k%d", i)
			kinds.Define(name, func(k world.Kind[body]) world.EntKind {
				return world.EntKind{
					Position: k.Load(func(b body) world.Position { return b.pos }),
					Velocity: k.Load(func(b body) world.Velocity { return b.vel }),
					Components: []world.ComponentTemplate{
						collisions.Collidable(wp.Space()),
						k.Const(hit.Mark{Duration: hitDuration}),
					},
				}
			})
			entries = append(entries, kinds.Entry(name, body{pos: placement.Place(i, count), vel: motion.initialVelocity(i)}))
		}
		return entries
	}
	wp.Seed(defineKinds(wp)...)
	if err := wp.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}
	cm := collisions.New(wp.Space(), ecs, 2*wp.MaxStep())

	ctx := &testInstallCtx{ecs: ecs}
	if err := wp.Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	var origIDs []uint64
	var origAppearance map[uint64]render.SpriteID
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems,
		goke.SystemFn{OnInit: func(si *goke.SysInit) {
			var posQ goke.Comp[world.Base]
			var appQ goke.Comp[world.Appearance]
			q := si.NewQueryBuilder(&posQ, &appQ).Build()
			origAppearance = make(map[uint64]render.SpriteID)
			q.All()
			for q.Next() {
				cur := q.Cursor()
				appearances := appQ.Slice(cur)
				for i, id := range cur.IDs {
					origIDs = append(origIDs, uint64(id))
					origAppearance[uint64(id)] = appearances[i].SpriteID
				}
			}
		}},
	)
	ecs.Setup(systems...)
	cm.RegSystems(ecs)

	if len(origIDs) != count {
		t.Fatalf("spawned %d entities, want %d", len(origIDs), count)
	}

	// Half one: collisions.Collidable ran as each entity was spawned.
	if got := countCollidable(wp.Space()); got != count {
		t.Errorf("%d of %d spawned entities can collide — the Collidable template did not register them", got, count)
	}

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	ecs.Resume()

	ecs2 := goke.New()
	plugin2 := world.NewPlugin(cfg)
	defineKinds(plugin2)
	cm2 := collisions.New(plugin2.Space(), ecs2, 2*plugin2.MaxStep())

	ctx2 := &testInstallCtx{ecs: ecs2}
	if err := plugin2.Install(ctx2); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Ask every installed module what it owns rather than re-listing it here:
	// a hand-written list silently rots the moment a plugin gains a component.
	comps := goke.ProvidedComps(append([]any{cm2}, ctx2.tracked...)...)
	if err := ecs2.Load(path, eachOnce(comps)...); err != nil {
		t.Fatalf("Load: %v", err)
	}
	cm2.RegSystems(ecs2)

	// cm2 goes in alongside the tracked values for the same reason it does
	// above: the engine tracks the collisions module through Plugin.Install,
	// which this test bypasses.
	var postLoad []goke.System
	for _, v := range append([]any{cm2}, ctx2.tracked...) {
		if pl, ok := v.(plugin.PostLoader); ok {
			postLoad = append(postLoad, pl.PostLoad())
		}
	}

	var loadedCount int
	postLoad = append(postLoad, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posQ goke.Comp[world.Base]
		var appQ goke.Comp[world.Appearance]
		q := si.NewQueryBuilder(&posQ, &appQ).Build()
		q.All()
		for q.Next() {
			cur := q.Cursor()
			appearances := appQ.Slice(cur)
			for i, id := range cur.IDs {
				wantSprite, ok := origAppearance[uint64(id)]
				if !ok {
					t.Errorf("entity %d: not among originally spawned IDs", id)
				} else if appearances[i].SpriteID != wantSprite {
					t.Errorf("entity %d: SpriteID = %d, want %d", id, appearances[i].SpriteID, wantSprite)
				}
				loadedCount++
			}
		}
	}})
	ecs2.Setup(postLoad...)

	if loadedCount != count {
		t.Fatalf("loaded %d entities, want %d", loadedCount, count)
	}

	// Half two: templates never ran here — a restored world spawns nothing —
	// so this is entirely collisions.module.PostLoad's doing.
	if got := countCollidable(plugin2.Space()); got != count {
		t.Errorf("%d of %d loaded entities can collide — PostLoad did not restore the capability", got, count)
	}
}

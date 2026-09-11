package main

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
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
	spawner := world.NewSpawner(
		func(index, count int) world.Position { return placement.Place(index, count) },
		func(index int) world.Velocity { return motion.initialVelocity(index) },
	).
		With(func(index int) world.Appearance {
			return world.Appearance{SpriteID: render.SpriteID(index)}
		}).
		With(func(index int) collisions.Collision { return collisions.Collision{} })
	wp.Populate(count, spawner)
	cm := collisions.New(wp.Space(), ecs, 0)

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
			var posQ goke.Comp[world.Position]
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

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	ecs.Resume()

	ecs2 := goke.New()
	plugin2 := world.NewPlugin(cfg)
	cm2 := collisions.New(plugin2.Space(), ecs2, 0)

	ctx2 := &testInstallCtx{ecs: ecs2}
	if err := plugin2.Install(ctx2); err != nil {
		t.Fatalf("Install: %v", err)
	}

	comps := append(goke.ProvidedComps(cm2),
		goke.LoadComp[world.Position](),
		goke.LoadComp[world.Appearance](),
		goke.LoadComp[world.Velocity](),
	)
	if err := ecs2.Load(path, comps...); err != nil {
		t.Fatalf("Load: %v", err)
	}
	cm2.RegSystems(ecs2)

	var postLoad []goke.System
	for _, v := range ctx2.tracked {
		if pl, ok := v.(plugin.PostLoader); ok {
			postLoad = append(postLoad, pl.PostLoad())
		}
	}

	var loadedCount int
	postLoad = append(postLoad, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posQ goke.Comp[world.Position]
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
}

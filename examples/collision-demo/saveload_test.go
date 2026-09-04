package main

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// TestSaveLoadCycle exercises the same mechanics Game.Save/Game.Load use, below the level of Game (no Ebiten window).
func TestSaveLoadCycle(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	const count = 5
	cfg := world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: count, MinSize: RectSize, MaxSize: RectSize},
	}

	ecs := goke.New()
	plugin := world.NewPlugin(cfg)
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
	plugin.Populate(count, spawner)
	cm := collisions.New(plugin.Space(), ecs)

	var origIDs []uint64
	var origAppearance map[uint64]render.SpriteID
	systems := append(plugin.Module().SetupSystems(),
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
	cm2 := collisions.New(plugin2.Space(), ecs2)

	comps := append(goke.ProvidedComps(cm2),
		goke.LoadComp[world.Position](),
		goke.LoadComp[world.Appearance](),
		goke.LoadComp[world.Velocity](),
	)
	if err := ecs2.Load(path, comps...); err != nil {
		t.Fatalf("Load: %v", err)
	}
	cm2.RegSystems(ecs2)

	var loadedCount int
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
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
	}}, plugin2.Module().PostLoad())

	if loadedCount != count {
		t.Fatalf("loaded %d entities, want %d", loadedCount, count)
	}
}

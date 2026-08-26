package main

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/grid"
	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/randomvelocity"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/world"
)

// TestSaveLoadCycle exercises the same mechanics Game.Save/Game.Load use, below the level of Game (no Ebiten window).
func TestSaveLoadCycle(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	const count = 5
	spaceCfg := world.Config{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true}
	pop := world.Population{MaxCount: count, MinSize: RectSize, MaxSize: RectSize}
	step := time.Second / TPS

	ecs := goke.New()
	wm := world.NewModule(spaceCfg, pop)
	wm.Populate(count, &kinematics.Telemetry{},
		kinematics.NewSpawner(grid.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize), randomvelocity.New(200, 50, 10)),
		render.NewAppearanceExtras(func(index int) render.Appearance {
			return render.Appearance{SpriteID: uint8(index)}
		}),
	)
	physicsModule := physics.New(wm.Space(), ecs, RectSize, step)

	var origIDs []uint64
	var origAppearance map[uint64]uint8
	systems := append(wm.SetupSystems(),
		goke.SystemFn{OnInit: func(si *goke.SysInit) {
			var posQ goke.Comp[kinematics.Position]
			var appQ goke.Comp[render.Appearance]
			q := si.NewQueryBuilder(&posQ, &appQ).Build()
			origAppearance = make(map[uint64]uint8)
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
	physicsModule.RegSystems(ecs)

	if len(origIDs) != count {
		t.Fatalf("spawned %d entities, want %d", len(origIDs), count)
	}

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	ecs.Resume()

	ecs2 := goke.New()
	wm2 := world.NewModule(spaceCfg, pop)
	physicsModule2 := physics.New(wm2.Space(), ecs2, RectSize, step)

	comps := append(goke.ProvidedComps(physicsModule2), goke.LoadComp[render.Appearance]())
	if err := ecs2.Load(path, comps...); err != nil {
		t.Fatalf("Load: %v", err)
	}
	physicsModule2.RegSystems(ecs2)

	var loadedCount int
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posQ goke.Comp[kinematics.Position]
		var appQ goke.Comp[render.Appearance]
		q := si.NewQueryBuilder(&posQ, &appQ).Build()
		q.All()
		for q.Next() {
			cur := q.Cursor()
			positions := posQ.Slice(cur)
			appearances := appQ.Slice(cur)
			for i, id := range cur.IDs {
				wm2.Space().Insert(id, positions[i].AABB)
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
	wm2.Space().Flush(nil)

	if loadedCount != count {
		t.Fatalf("loaded %d entities, want %d", loadedCount, count)
	}
}

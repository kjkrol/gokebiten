package gokebiten_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/render"
)

type saveTestState struct{ N int }
type saveTestResourceB struct{ S string }

// ecsAccessor captures ctx.ECS() during Install — the only way to reach *goke.ECS now that Game.ECS() is gated behind Plugin.
type ecsAccessor struct{ ecs *goke.ECS }

func (a *ecsAccessor) Name() string { return "test.ecs-accessor" }
func (a *ecsAccessor) Install(ctx *gokebiten.GameCtx) error {
	a.ecs = ctx.ECS()
	return nil
}

// TestGame_SaveLoad_RoundTrip guards that Game.Persistence.Save/Load correctly delegate to the game's own ECS and resources.
func TestGame_SaveLoad_RoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"

	game := gokebiten.NewGame(&gokebiten.GameProps{})
	state := &saveTestState{N: 42}
	extra := &saveTestResourceB{S: "hello"}

	acc := &ecsAccessor{}
	if err := game.UsePlugin(acc); err != nil {
		t.Fatalf("UsePlugin(ecsAccessor): %v", err)
	}

	var appearance goke.Comp[render.Appearance]
	acc.ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&appearance)
		f.Create(1)
		f.Next()
		appearance.Slice(&f.Cursor)[0] = render.Appearance{SpriteID: 7}
	}})

	if err := game.Persistence.Save(basePath, "", state, extra); err != nil {
		t.Fatalf("Save: %v", err)
	}

	game2 := gokebiten.NewGame(&gokebiten.GameProps{})
	acc2 := &ecsAccessor{}
	if err := game2.UsePlugin(acc2); err != nil {
		t.Fatalf("UsePlugin(ecsAccessor): %v", err)
	}
	state2 := &saveTestState{}
	extra2 := &saveTestResourceB{}

	if err := game2.Persistence.Load(basePath, "", state2, extra2); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state2.N != 42 {
		t.Errorf("state2.N after Load = %d, want 42", state2.N)
	}
	if extra2.S != "hello" {
		t.Errorf("extra2.S after Load = %q, want %q", extra2.S, "hello")
	}

	var appearance2 goke.Comp[render.Appearance]
	var q *goke.Query
	acc2.ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&appearance2).Build()
	}})

	q.All()
	found := false
	for q.Next() {
		for _, a := range appearance2.Slice(q.Cursor()) {
			found = true
			if a.SpriteID != 7 {
				t.Errorf("SpriteID = %d, want 7", a.SpriteID)
			}
		}
	}
	if !found {
		t.Fatal("expected the saved render.Appearance entity to survive the round trip")
	}
}

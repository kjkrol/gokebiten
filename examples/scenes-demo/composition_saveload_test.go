package main

import (
	"testing"

	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func testProps() game.Props {
	return game.Props{
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		World: world.Config{
			Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
			Entities: world.EntitiesCfg{MaxCount: EntityCount, MinSize: EntitySize, MaxSize: EntitySize},
		},
	}
}

// oneStageGame is a minimal game.Game wrapping a single Stage.
type oneStageGame struct {
	stage game.Stage
	props game.Props
}

func (g oneStageGame) Props() game.Props { return g.props }

func (g oneStageGame) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{g.stage.Name(): g.stage}, g.stage.Name()
}

// TestGameplayStage_Composition_SurvivesSaveLoad guards the engine-level
// persistence mechanism Composition relies on: opening the panel, saving,
// then loading into a fresh Stage/Engine restores the same visible order
// and the same active scene — with no save/load code written in
// GameplayStage itself beyond the one ctx.Track(g.composition) call in Init.
func TestGameplayStage_Composition_SurvivesSaveLoad(t *testing.T) {
	basePath := t.TempDir() + "/save"

	stage := &GameplayStage{SaveBasePath: basePath}
	eng := engine.NewEngine(oneStageGame{stage: stage, props: testProps()})
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	stage.Composition().Show(stage.panel.Name())
	if got, want := stage.Composition().Active(), stage.panel.Name(); got != want {
		t.Fatalf("Active() before save = %q, want %q", got, want)
	}

	if err := eng.Persistence().Save(basePath, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	stage2 := &GameplayStage{SaveBasePath: basePath}
	eng2 := engine.NewEngine(oneStageGame{stage: stage2, props: testProps()})
	if err := eng2.Init(); err != nil {
		t.Fatalf("Init (fresh process/engine): %v", err)
	}

	wantOrder := []string{"world", "hud", "panel"}
	if got := stage2.Composition().Order(); !equalStrings(got, wantOrder) {
		t.Errorf("Order() after Load = %v, want %v", got, wantOrder)
	}
	if got, want := stage2.Composition().Active(), stage2.panel.Name(); got != want {
		t.Errorf("Active() after Load = %q, want %q (the panel should still be on top and focused)", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

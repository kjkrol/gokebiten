package engine

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gram/game"
)

func TestInit_MakesTheEnginesStepTheOnlyClock(t *testing.T) {
	ebiten.SetTPS(60)
	eng := newTestEngine(func(game.Initializer) error { return nil })
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := ebiten.TPS(); got != ebiten.SyncWithFPS {
		t.Errorf("Ebitengine TPS = %d after Init, want SyncWithFPS: one Update per frame, the ticks paced by the engine", got)
	}
}

func TestTracker_AFrameFarBehindRunsAtMostMaxStepsAndDropsTheRest(t *testing.T) {
	const step = time.Second / 120
	tr := newTracker()
	tr.lastUpdate = time.Now().Add(-time.Second)

	if steps := tr.calculateSteps(step, 5); steps != 5 {
		t.Fatalf("a frame a second behind ran %d steps, want the cap of 5", steps)
	}
	if steps := tr.calculateSteps(step, 5); steps > 1 {
		t.Errorf("the next frame ran %d steps, want the dropped second not to be caught up", steps)
	}
}

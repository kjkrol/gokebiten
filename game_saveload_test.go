package gokebiten

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/spatial"
)

type saveTestState struct{ N int }

func (s saveTestState) MarshalBinary() ([]byte, error)     { return []byte{byte(s.N)}, nil }
func (s *saveTestState) UnmarshalBinary(data []byte) error { s.N = int(data[0]); return nil }

type saveTestTelemetry struct{}

func testSpace(t *testing.T) *spatial.WorldModule {
	t.Helper()
	return spatial.NewWorldModule(
		spatial.Config{Width: 100, Height: 100},
		spatial.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)
}

// TestGame_SaveLoad_RoundTrip guards Save/Load's core promise: ONE file on
// disk holding both State and the ECS snapshot (spliced via a temp file,
// since goke's ECS.Save/Load only take paths — see the conversation this
// was designed in), with no leftover temp files and no stray second file.
func TestGame_SaveLoad_RoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"

	res := NewResources(&GameProps{}, spatial.Config{}, saveTestState{N: 42}, saveTestTelemetry{})
	game := NewGame(res)

	var appearance goke.Comp[render.Appearance]
	game.ECS().Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&appearance)
		f.Create(1)
		f.Next()
		appearance.Slice(&f.Cursor)[0] = render.Appearance{SpriteID: 7}
	}})

	if err := game.Save(basePath, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantPath := basePath + ".game.save"
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected %s to exist: %v", wantPath, err)
	}
	if matches, _ := filepath.Glob(basePath + "*"); len(matches) != 1 {
		t.Fatalf("expected exactly one save file for basePath, got %v", matches)
	}
	if tmps, _ := filepath.Glob(filepath.Join(os.TempDir(), "gokebiten-ecs-*.tmp")); len(tmps) != 0 {
		t.Errorf("expected no leftover temp files after Save, got %v", tmps)
	}

	res2 := NewResources(&GameProps{}, spatial.Config{}, saveTestState{}, saveTestTelemetry{})
	game2 := NewGame(res2)
	world2 := testSpace(t)

	var loadedCount int
	onLoaded := func(count int) { loadedCount = count }
	if err := game2.Load(basePath, "", world2.Space(), onLoaded); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tmps, _ := filepath.Glob(filepath.Join(os.TempDir(), "gokebiten-ecs-*.tmp")); len(tmps) != 0 {
		t.Errorf("expected no leftover temp files after Load, got %v", tmps)
	}

	var appearance2 goke.Comp[render.Appearance]
	var q *goke.Query
	game2.ECS().Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&appearance2).Build()
	}})

	if loadedCount != 0 {
		t.Errorf("loadedCount (kinematics.Position entities) = %d, want 0 (this test only seeded a render.Appearance)", loadedCount)
	}
	if res2.State().N != 42 {
		t.Errorf("State().N after Load = %d, want 42", res2.State().N)
	}

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

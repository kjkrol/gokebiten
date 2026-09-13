package render

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestAtlas_RegisterAt_BindsPixelsToTheGivenSlot(t *testing.T) {
	first, second := SpriteID(0), SpriteID(1)

	atlas := NewAtlas(4, 2)
	var drawnFor []SpriteID
	atlas.RegisterAt(second, func(dst *ebiten.Image, size int) { drawnFor = append(drawnFor, second) })
	atlas.RegisterAt(first, func(dst *ebiten.Image, size int) { drawnFor = append(drawnFor, first) })
	atlas.Close()

	if len(drawnFor) != 2 {
		t.Fatalf("expected both drawers to run, got %v", drawnFor)
	}

	sx0, _, sx1, _ := atlas.UV(second)
	if want := float32(4); sx0 != want || sx1 != want+4 {
		t.Errorf("UV(second) = [%v,%v], want slot at x=%v", sx0, sx1, want)
	}
}

func TestAtlas_RegisterAt_PanicsOutOfRange(t *testing.T) {
	atlas := NewAtlas(4, 1)
	defer func() {
		if recover() == nil {
			t.Fatal("expected RegisterAt to panic for an id beyond capacity")
		}
	}()
	atlas.RegisterAt(SpriteID(1), func(*ebiten.Image, int) {})
}

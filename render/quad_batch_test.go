package render

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/camera"
)

type fakeAtlasSource struct{}

func (fakeAtlasSource) Atlas() *ebiten.Image { return nil }
func (fakeAtlasSource) UV(SpriteID) (float32, float32, float32, float32) {
	return 0, 0, 1, 1
}

// TestQuadBatch_AppendQuad_ConsistentAcrossWrapSeam guards that a quad
// straddling a toroidal camera's wrap reference point splits into pieces
// that keep its true (small) total size — projecting each corner through
// ToScreen independently, or clipping the whole quad to one side, either
// stretches or drops part of it instead.
func TestQuadBatch_AppendQuad_ConsistentAcrossWrapSeam(t *testing.T) {
	cam := camera.NewFromSpace(1024, 1024, true)
	cam.Translate(1000, 0) // effective.TopLeft.X is now 1000 — the wrap reference

	batch := NewQuadBatch(fakeAtlasSource{}, cam)
	batch.AppendQuad(998, 0, 1010, 10, 0) // straddles world x=1000: splits into widths 2 and 10

	if len(batch.vertices) != 8 {
		t.Fatalf("len(vertices) = %d, want 8 (two split quads)", len(batch.vertices))
	}
	firstWidth := batch.vertices[1].DstX - batch.vertices[0].DstX
	secondWidth := batch.vertices[5].DstX - batch.vertices[4].DstX
	if firstWidth+secondWidth != 12 {
		t.Errorf("total quad width straddling the wrap seam = %v, want 12 (must not stretch or drop part of it)", firstWidth+secondWidth)
	}
}

package render

import (
	"github.com/kjkrol/aabbworld"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gram/camera"
)

type fakeAtlasSource struct{}

func (fakeAtlasSource) Atlas() *ebiten.Image { return nil }
func (fakeAtlasSource) UV(SpriteID) (float32, float32, float32, float32) {
	return 0, 0, 1, 1
}

func TestQuadBatch_AppendQuad_ConsistentAcrossWrapSeam(t *testing.T) {
	cam := camera.NewFromSpace(1024, 1024, aabbworld.Torus)
	cam.Translate(1000, 0)

	batch := NewQuadBatch(fakeAtlasSource{}, cam)
	batch.AppendQuad(998, 0, 1010, 10, 0)

	if len(batch.vertices) != 8 {
		t.Fatalf("len(vertices) = %d, want 8 (two split quads)", len(batch.vertices))
	}
	firstWidth := batch.vertices[1].DstX - batch.vertices[0].DstX
	secondWidth := batch.vertices[5].DstX - batch.vertices[4].DstX
	if firstWidth+secondWidth != 12 {
		t.Errorf("total quad width straddling the wrap seam = %v, want 12 (must not stretch or drop part of it)", firstWidth+secondWidth)
	}
}

func TestQuadBatch_IndicesRestartPerChunk(t *testing.T) {
	cam := camera.NewFromSpace(100000, 100, 0)
	batch := NewQuadBatch(fakeAtlasSource{}, cam)
	quads := chunkVertices/4 + 5
	for i := range quads {
		x := float32(i)
		batch.AppendQuad(x, 0, x+1, 1, 0)
	}
	if len(batch.vertices) != quads*4 || len(batch.indices) != quads*6 {
		t.Fatalf("%d vertices and %d indices for %d quads", len(batch.vertices), len(batch.indices), quads)
	}
	first := batch.indices[chunkVertices/4*6]
	if first != 0 {
		t.Errorf("the first index past the chunk is %d, want 0 (relative to its own DrawTriangles call)", first)
	}
	for i, idx := range batch.indices {
		if int(idx) >= chunkVertices {
			t.Fatalf("index %d = %d reaches past a chunk", i, idx)
		}
	}
}

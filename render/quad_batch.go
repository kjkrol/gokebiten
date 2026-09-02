package render

import "github.com/hajimehoshi/ebiten/v2"

// QuadBatch batches textured quads from an AtlasSource into a single
// DrawTriangles call, transformed through a Camera.
type QuadBatch struct {
	atlas    AtlasSource
	camera   Camera
	vertices []ebiten.Vertex
	indices  []uint16
	triOpts  *ebiten.DrawTrianglesOptions
}

func NewQuadBatch(atlas AtlasSource) *QuadBatch {
	return &QuadBatch{atlas: atlas, triOpts: &ebiten.DrawTrianglesOptions{}}
}

func (b *QuadBatch) BindCamera(camera Camera) { b.camera = camera }

func (b *QuadBatch) Reset() { b.vertices = b.vertices[:0]; b.indices = b.indices[:0] }

// AppendQuadUV appends one screen-space quad for the world-space box
// (x0,y0)-(x1,y1), sampling the [u0,v0]-[u1,v1] fractional sub-rect of
// sprite id's own UV region (0,0 = sprite's top-left, 1,1 = bottom-right) —
// for a partial slice of the sprite (e.g. a toroidal-wrap fragment), not
// the whole thing.
func (b *QuadBatch) AppendQuadUV(x0, y0, x1, y1 float32, id SpriteID, u0, v0, u1, v1 float32) {
	sx0, sy0, sx1, sy1 := b.atlas.UV(id)
	spriteW, spriteH := sx1-sx0, sy1-sy0
	fsx0, fsy0 := sx0+u0*spriteW, sy0+v0*spriteH
	fsx1, fsy1 := sx0+u1*spriteW, sy0+v1*spriteH

	x0, y0 = b.camera.ToScreen(x0, y0)
	x1, y1 = b.camera.ToScreen(x1, y1)
	idx := uint16(len(b.vertices))
	b.vertices = append(b.vertices,
		ebiten.Vertex{DstX: x0, DstY: y0, SrcX: fsx0, SrcY: fsy0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: x1, DstY: y0, SrcX: fsx1, SrcY: fsy0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: x0, DstY: y1, SrcX: fsx0, SrcY: fsy1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: x1, DstY: y1, SrcX: fsx1, SrcY: fsy1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	)
	b.indices = append(b.indices, idx, idx+1, idx+2, idx+1, idx+2, idx+3)
}

// AppendQuad is AppendQuadUV over the sprite's full region — the common case (no slicing).
func (b *QuadBatch) AppendQuad(x0, y0, x1, y1 float32, id SpriteID) {
	b.AppendQuadUV(x0, y0, x1, y1, id, 0, 0, 1, 1)
}

func (b *QuadBatch) Flush(screen *ebiten.Image) {
	if len(b.indices) > 0 {
		screen.DrawTriangles(b.vertices, b.indices, b.atlas.Atlas(), b.triOpts)
	}
}

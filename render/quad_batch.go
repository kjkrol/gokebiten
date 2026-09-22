package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/camera"
)

// QuadBatch batches textured quads from an AtlasSource into a single
// DrawTriangles call, transformed through a camera.Camera.
type QuadBatch struct {
	atlas    AtlasSource
	camera   camera.Camera
	vertices []ebiten.Vertex
	indices  []uint16
	quads    []camera.Quad
	triOpts  *ebiten.DrawTrianglesOptions
}

func NewQuadBatch(atlas AtlasSource, cam camera.Camera) *QuadBatch {
	return &QuadBatch{atlas: atlas, camera: cam, triOpts: &ebiten.DrawTrianglesOptions{}}
}

func (b *QuadBatch) Reset() { b.vertices = b.vertices[:0]; b.indices = b.indices[:0] }

// AppendQuadUV appends a quad for the world box (x0,y0)-(x1,y1), sampling id's UV sub-rect.
func (b *QuadBatch) AppendQuadUV(x0, y0, x1, y1 float32, id SpriteID, u0, v0, u1, v1 float32) {
	sx0, sy0, sx1, sy1 := b.atlas.UV(id)
	spriteW, spriteH := sx1-sx0, sy1-sy0

	b.quads = b.camera.ToScreenQuads(x0, y0, x1, y1, b.quads[:0])
	for _, q := range b.quads {
		pu0, pu1 := u0+q.T0X*(u1-u0), u0+q.T1X*(u1-u0)
		pv0, pv1 := v0+q.T0Y*(v1-v0), v0+q.T1Y*(v1-v0)
		fsx0, fsy0 := sx0+pu0*spriteW, sy0+pv0*spriteH
		fsx1, fsy1 := sx0+pu1*spriteW, sy0+pv1*spriteH

		idx := uint16(len(b.vertices))
		b.vertices = append(b.vertices,
			ebiten.Vertex{DstX: q.X0, DstY: q.Y0, SrcX: fsx0, SrcY: fsy0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: q.X1, DstY: q.Y0, SrcX: fsx1, SrcY: fsy0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: q.X0, DstY: q.Y1, SrcX: fsx0, SrcY: fsy1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: q.X1, DstY: q.Y1, SrcX: fsx1, SrcY: fsy1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		)
		b.indices = append(b.indices, idx, idx+1, idx+2, idx+1, idx+2, idx+3)
	}
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

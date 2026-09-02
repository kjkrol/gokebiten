package world

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

const (
	fragScreenLeft    = plane.FRAG_RIGHT
	fragScreenTop     = plane.FRAG_BOTTOM
	fragScreenTopLeft = plane.FRAG_BOTTOM_RIGHT
)

// spriteBatch is render.QuadBatch plus toroidal-fragment-aware slicing —
// a wrapped entity draws only the sliver of its sprite that actually
// crossed the world edge, not a full duplicate.
type spriteBatch struct {
	batch  *render.QuadBatch
	camera render.Camera
}

func newSpriteBatch(atlas render.AtlasSource) spriteBatch {
	return spriteBatch{batch: render.NewQuadBatch(atlas)}
}

func (b *spriteBatch) bindCamera(camera render.Camera) {
	b.camera = camera
	b.batch.BindCamera(camera)
}

func (b *spriteBatch) reset() { b.batch.Reset() }

func (b *spriteBatch) drawQuad(pos Position, id render.SpriteID) {
	if !b.camera.Visible(pos.AABB.AABB) {
		return
	}
	sizeX := float32(pos.AABB.Size.X)
	sizeY := float32(pos.AABB.Size.Y)

	pos.AABB.VisitFragments(func(fp plane.FragPosition, fragBox geom.AABB[uint32]) bool {
		tlx, tly := float32(fragBox.TopLeft.X), float32(fragBox.TopLeft.Y)
		brx, bry := float32(fragBox.BottomRight.X), float32(fragBox.BottomRight.Y)
		fw, fh := brx-tlx, bry-tly

		var u0, v0, u1, v1 float32 = 0, 0, 1, 1
		switch fp {
		case fragScreenLeft: // FRAG_RIGHT
			u0 = 1 - fw/sizeX
		case fragScreenTop: // FRAG_BOTTOM
			v0 = 1 - fh/sizeY
		case fragScreenTopLeft: // FRAG_BOTTOM_RIGHT
			u0, v0 = 1-fw/sizeX, 1-fh/sizeY
		default:
			return true
		}
		b.batch.AppendQuadUV(tlx, tly, brx, bry, id, u0, v0, u1, v1)
		return true
	})

	mainBox := pos.AABB.AABB
	tlx, tly := float32(mainBox.TopLeft.X), float32(mainBox.TopLeft.Y)
	brx, bry := float32(mainBox.BottomRight.X), float32(mainBox.BottomRight.Y)
	b.batch.AppendQuadUV(tlx, tly, brx, bry, id, 0, 0, (brx-tlx)/sizeX, (bry-tly)/sizeY)
}

func (b *spriteBatch) flush(screen *ebiten.Image) { b.batch.Flush(screen) }

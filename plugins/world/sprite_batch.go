package world

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

// spriteBatch is render.QuadBatch plus toroidal-fragment-aware slicing —
// a wrapped entity draws only the sliver of its sprite that actually
// crossed the world edge, not a full duplicate.
type spriteBatch struct {
	batch  *render.QuadBatch
	camera camera.Camera
	worldW float32
	worldH float32
}

func newSpriteBatch(cam camera.Camera, atlas render.AtlasSource, worldW, worldH uint32) spriteBatch {
	return spriteBatch{
		batch:  render.NewQuadBatch(atlas, cam),
		camera: cam,
		worldW: float32(worldW),
		worldH: float32(worldH),
	}
}

func (b *spriteBatch) reset() { b.batch.Reset() }

func (b *spriteBatch) drawQuad(pos Position, id render.SpriteID) {
	if !b.camera.Visible(pos.AABB.AABB) {
		return
	}
	sizeX := float32(pos.AABB.Size.X)
	sizeY := float32(pos.AABB.Size.Y)

	render.VisitWrapImages(pos.AABB, b.worldW, b.worldH, func(img geom.AABB, dx, dy float32) bool {
		tlx, tly := float32(img.TopLeft.X), float32(img.TopLeft.Y)
		brx, bry := float32(img.BottomRight.X), float32(img.BottomRight.Y)

		u0, u1 := uvSpan(brx-tlx, sizeX, dx)
		v0, v1 := uvSpan(bry-tly, sizeY, dy)
		b.batch.AppendQuadUV(tlx, tly, brx, bry, id, u0, v0, u1, v1)
		return true
	})
}

// uvSpan is the slice of the sprite one image shows along one axis. An image
// sitting at no shift holds the head of the sprite up to where the world edge
// cut it; one shifted a world back holds the tail that carried on past it.
//
// Both axes are decided separately, which matters for an entity straddling the
// world's corner: its side and bottom pieces are each clipped on the other axis
// too, and stretching the whole sprite across them would smear it.
func uvSpan(imgSize, spriteSize, shift float32) (float32, float32) {
	visible := imgSize / spriteSize
	if shift == 0 {
		return 0, visible
	}
	return 1 - visible, 1
}

func (b *spriteBatch) flush(screen *ebiten.Image) { b.batch.Flush(screen) }

package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// SpriteID identifies a pre-baked sprite in an Atlas — resolved by array
// index (Atlas.UV), never by map lookup, so it's safe on the draw hot path.
type SpriteID uint8

// SpriteDrawer paints one sprite's size x size pixels into dst — passed to
// Atlas.Register to bake a new sprite.
type SpriteDrawer func(dst *ebiten.Image, size int)

// AtlasSource supplies the sprite sheet and per-sprite UV rects that
// QuadBatch draws quads from — swappable, e.g. procedurally-drawn shapes
// vs. sprites loaded from a file.
type AtlasSource interface {
	Atlas() *ebiten.Image
	UV(id SpriteID) (sx0, sy0, sx1, sy1 float32)
}

// Atlas is a fixed-capacity AtlasSource built by registering one sprite at
// a time (Register), then freezing (Close) before the game loop starts —
// UV lookups are always O(1) slice indexing, never a map.
type Atlas struct {
	image      *ebiten.Image
	spriteSize int
	capacity   int
	count      int
	closed     bool
}

var _ AtlasSource = (*Atlas)(nil)

// NewAtlas allocates an atlas of capacity sprites, each spriteSize x spriteSize texels.
func NewAtlas(spriteSize, capacity int) *Atlas {
	return &Atlas{
		image:      ebiten.NewImage(spriteSize*capacity, spriteSize),
		spriteSize: spriteSize,
		capacity:   capacity,
	}
}

// Register bakes draw's output into the next free slot and returns its SpriteID — panics if Close was already called or capacity is exhausted.
func (a *Atlas) Register(draw SpriteDrawer) SpriteID {
	if a.closed {
		panic("gokebiten: Atlas.Register after Close")
	}
	if a.count >= a.capacity {
		panic(fmt.Sprintf("gokebiten: Atlas capacity %d exhausted", a.capacity))
	}
	sprite := ebiten.NewImage(a.spriteSize, a.spriteSize)
	draw(sprite, a.spriteSize)
	id := SpriteID(a.count)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(int(id)*a.spriteSize), 0)
	a.image.DrawImage(sprite, opts)
	a.count++
	return id
}

// Close freezes the atlas — call once, after every Register, before the game loop starts.
func (a *Atlas) Close() { a.closed = true }

func (a *Atlas) Atlas() *ebiten.Image { return a.image }

func (a *Atlas) UV(id SpriteID) (sx0, sy0, sx1, sy1 float32) {
	sx0 = float32(int(id) * a.spriteSize)
	sy0 = 0
	sx1 = sx0 + float32(a.spriteSize)
	sy1 = float32(a.spriteSize)
	return
}

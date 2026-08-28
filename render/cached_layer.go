package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
)

// CachedLayer draws inner once into an offscreen image and reuses it every
// frame — call Invalidate to force a redraw on the next Draw.
type CachedLayer struct {
	inner Layer
	image *ebiten.Image
	dirty bool
}

// NewCachedLayer wraps inner, caching its output at w×h until invalidated.
func NewCachedLayer(inner Layer, w, h int) *CachedLayer {
	return &CachedLayer{inner: inner, image: ebiten.NewImage(w, h), dirty: true}
}

func (c *CachedLayer) Init(si *goke.SysInit) { c.inner.Init(si) }

// Invalidate forces the next Draw to redraw the cached image.
func (c *CachedLayer) Invalidate() { c.dirty = true }

func (c *CachedLayer) Draw(screen *ebiten.Image) {
	if c.dirty {
		c.image.Clear()
		c.inner.Draw(c.image)
		c.dirty = false
	}
	screen.DrawImage(c.image, nil)
}

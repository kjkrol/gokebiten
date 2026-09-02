package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Solid returns a SpriteDrawer filling the whole sprite with c.
func Solid(c color.RGBA) SpriteDrawer {
	return func(dst *ebiten.Image, size int) {
		vector.FillRect(dst, 0, 0, float32(size), float32(size), c, false)
	}
}

// Border returns a SpriteDrawer outlining the sprite with c, transparent inside.
func Border(c color.RGBA) SpriteDrawer {
	return func(dst *ebiten.Image, size int) {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if x <= 1 || x >= size-2 || y <= 1 || y >= size-2 {
					dst.Set(x, y, c)
				}
			}
		}
	}
}

// Diamond returns a SpriteDrawer filling a diamond shape with c, transparent outside.
func Diamond(c color.RGBA) SpriteDrawer {
	return func(dst *ebiten.Image, size int) {
		cx, cy := size/2, size/2
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if abs(x-cx)+abs(y-cy) <= cx-1 {
					dst.Set(x, y, c)
				}
			}
		}
	}
}

// Cross returns a SpriteDrawer filling a cross shape with c, transparent outside.
func Cross(c color.RGBA) SpriteDrawer {
	return func(dst *ebiten.Image, size int) {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				inH := y >= size/4 && y < size*3/4
				inV := x >= size/4 && x < size*3/4
				if inH || inV {
					dst.Set(x, y, c)
				}
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

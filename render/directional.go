package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func endpointAt(size int, angleDeg float64) (x, y float32) {
	cx, cy := float32(size)/2, float32(size)/2
	rad := angleDeg * math.Pi / 180
	dx, dy := float32(math.Cos(rad)), -float32(math.Sin(rad))
	half := float32(size) / 2
	scale := half / max(abs32(dx), abs32(dy))
	return cx + dx*scale, cy + dy*scale
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func Arrow(angleDeg float64, width float32, c color.RGBA) SpriteDrawer {
	return func(dst *ebiten.Image, size int) {
		x0, y0 := endpointAt(size, angleDeg)
		cx, cy := float32(size)/2, float32(size)/2
		vector.StrokeLine(dst, x0, y0, cx, cy, width, c, true)
	}
}

func Dot(radius float32, c color.RGBA) SpriteDrawer {
	return func(dst *ebiten.Image, size int) {
		vector.FillCircle(dst, float32(size)/2, float32(size)/2, radius, c, true)
	}
}

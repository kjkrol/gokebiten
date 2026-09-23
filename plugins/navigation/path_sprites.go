package navigation

import (
	"image/color"

	"github.com/kjkrol/gram/render"
)

// PathSprites is what a route is drawn from: a spoke per Direction and a dot for its end.
type PathSprites struct {
	Spokes [Directions]render.SpriteID
	Dot    render.SpriteID
}

func (s PathSprites) spoke(d Direction) render.SpriteID { return s.Spokes[d%Directions] }

// RegisterDefaultPathSprites builds an atlas of arrows in c, one per Direction, and a dot.
func RegisterDefaultPathSprites(spriteSize int, strokeWidth float32, c color.RGBA) (*render.Atlas, PathSprites) {
	atlas := render.NewAtlas()
	var sprites PathSprites
	for d := range Direction(Directions) {
		sprites.Spokes[d] = atlas.Register(spriteSize, render.Arrow(d.Angle(), strokeWidth, c))
	}
	sprites.Dot = atlas.Register(spriteSize, render.Dot(strokeWidth*2, c))
	atlas.Close()
	return atlas, sprites
}

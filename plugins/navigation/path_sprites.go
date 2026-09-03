package navigation

import (
	"image/color"

	"github.com/kjkrol/gokebiten/render"
)

type PathSprites struct {
	N, S, E, W, NE, NW, SE, SW render.SpriteID
	Dot                        render.SpriteID
}

func (s PathSprites) spoke(d Direction) render.SpriteID {
	switch d {
	case DirN:
		return s.N
	case DirS:
		return s.S
	case DirE:
		return s.E
	case DirW:
		return s.W
	case DirNE:
		return s.NE
	case DirNW:
		return s.NW
	case DirSE:
		return s.SE
	default:
		return s.SW
	}
}

func directionAngle(d Direction) float64 {
	switch d {
	case DirE:
		return 0
	case DirNE:
		return 45
	case DirN:
		return 90
	case DirNW:
		return 135
	case DirW:
		return 180
	case DirSW:
		return 225
	case DirS:
		return 270
	default:
		return 315
	}
}

const pathSpriteCount = 9

func RegisterDefaultPathSprites(spriteSize int, strokeWidth float32, c color.RGBA) (*render.Atlas, PathSprites) {
	atlas := render.NewAtlas(spriteSize, pathSpriteCount)

	spoke := func(d Direction) render.SpriteID {
		return atlas.Register(render.Arrow(directionAngle(d), strokeWidth, c))
	}

	sprites := PathSprites{
		N: spoke(DirN), S: spoke(DirS), E: spoke(DirE), W: spoke(DirW),
		NE: spoke(DirNE), NW: spoke(DirNW), SE: spoke(DirSE), SW: spoke(DirSW),
		Dot: atlas.Register(render.Dot(strokeWidth*2, c)),
	}

	atlas.Close()
	return atlas, sprites
}

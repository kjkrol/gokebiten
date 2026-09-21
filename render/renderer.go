package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
)

// Renderer draws each frame; Init runs once, at registration.
type Renderer interface {
	Init(*goke.SysInit)
	Draw(screen *ebiten.Image)
}

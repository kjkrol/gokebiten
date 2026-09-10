package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
)

// Renderer draws each frame; Init runs once at registration, outside the
// per-tick Update cycle — there is no scheduled Update here. Renderers that
// need a camera.Camera take it as a constructor argument.
type Renderer interface {
	Init(*goke.SysInit)
	Draw(screen *ebiten.Image)
}

package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
)

// Layer draws each frame; Init runs once at registration, outside the
// per-tick Update cycle — there is no scheduled Update here.
type Layer interface {
	Init(*goke.SysInit)
	Draw(screen *ebiten.Image)
}

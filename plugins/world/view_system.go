package world

import (
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
)

var _ goke.System = (*ViewSystem)(nil)

// ViewSystem refreshes every View the world keeps, from the Space as it stands after this tick's
// movement: the bounds are read anew, and the entities in them marked. A View whose bounds cover
// the whole world is not queried; it simply sees everything.
type ViewSystem struct {
	space     *aabbworld.Space
	views     *[]*View
	worldArea float64
}

// NewViewSystem builds the system over space, refreshing the Views listed at views.
func NewViewSystem(space *aabbworld.Space, views *[]*View, worldW, worldH uint32) *ViewSystem {
	return &ViewSystem{space: space, views: views, worldArea: float64(worldW) * float64(worldH)}
}

func (s *ViewSystem) Init(*goke.SysInit) {}

func (s *ViewSystem) Update(*goke.CmdBuf, time.Duration) {
	for _, v := range *s.views {
		s.refresh(v)
	}
}

func (s *ViewSystem) refresh(v *View) {
	b := v.bounds()
	v.Bounds = b
	v.In.Clear()
	if (b.BottomRight.X-b.TopLeft.X)*(b.BottomRight.Y-b.TopLeft.Y) >= s.worldArea {
		v.Culled = false
		return
	}
	v.Culled = true
	s.space.Query(b, aabbworld.AnyCapability, v.add)
}

package world

import (
	"testing"

	"github.com/kjkrol/gokebiten/render"
)

// Facing reads the heading out of the Base its host hands over — the renderer's
// query already requires Base, and a query refuses to carry a component twice.
func TestFacing_PicksTheSpriteFromTheEntitysHeading(t *testing.T) {
	const eastward, other render.SpriteID = 7, 3
	modifier := Facing(func(v Velocity) render.SpriteID {
		if v.Dir.X > 0 {
			return eastward
		}
		return other
	})

	for heading, want := range map[string]struct {
		dir  Velocity
		want render.SpriteID
	}{
		"east":  {Velocity{Dir: east, Value: 1}, eastward},
		"north": {Velocity{Dir: north, Value: 1}, other},
	} {
		base := Base{Vel: want.dir}
		if got := modifier.Apply(nil, 0, &base, []Appearance{{}})[0].SpriteID; got != want.want {
			t.Errorf("heading %s: SpriteID = %v, want %v", heading, got, want.want)
		}
	}
}

package vision_test

import (
	"math"
	"strings"
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugins/vision"
	"github.com/kjkrol/gram/plugins/world"
)

func z(altitude, height float64) *world.Z { return &world.Z{Altitude: altitude, Height: height} }

// eyed is an eastward Sight looking from eye above its bottom.
func eyed(eye float64) *vision.Sight {
	return &vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: math.Pi / 8, Radius: 300, Eye: eye}
}

// A wall 10 tall 100 ahead and a walker 240 ahead: a walker looking from 1.5 sees the wall alone,
// a hawk 40 up sees over it.
func TestHeights_AWalkerIsStoppedByTheWallAHawkLooksOver(t *testing.T) {
	quasi := &relief{}
	wallAndWalker := []spawn{
		{x: 100, y: 0, size: 60, z: z(0, 10)},
		{x: 240, y: 0, z: z(0, 2)},
	}
	_, seen, _ := sceneIn(t, quasi, append([]spawn{{x: 0, y: 0, z: z(0, 2), sight: eyed(1.5)}}, wallAndWalker...)...)
	if seen[0].Count != 1 || seen[0].Dists[0] > 100 {
		t.Errorf("the walker saw %d at %v, want the wall alone", seen[0].Count, seen[0].Dists[:seen[0].Count])
	}
	_, seen, _ = sceneIn(t, quasi, append([]spawn{{x: 0, y: 0, z: z(40, 2), sight: eyed(1)}}, wallAndWalker...)...)
	if seen[0].Count != 2 {
		t.Errorf("the hawk saw %d, want the wall and the walker behind it", seen[0].Count)
	}
}

// plateau is ground 12 high for x in [50, 100), 0 elsewhere.
type plateau struct{}

func (plateau) At(p geom.Vec) float64 {
	if p.X >= 50 && p.X < 100 {
		return 12
	}
	return 0
}
func (plateau) Step() float64 { return 10 }

func TestHeights_AHillHidesTheLowlandFromAWalkerAndNotFromAHawk(t *testing.T) {
	onHill := &relief{ground: plateau{}, step: 10}
	target := spawn{x: 240, y: 0, z: z(0, 2)}
	_, seen, outlines := sceneIn(t, onHill, spawn{x: 0, y: 0, z: z(0, 2), sight: eyed(1.5), outline: true}, target)
	if seen[0].Count != 0 {
		t.Errorf("the walker saw %d past the hill, want nothing", seen[0].Count)
	}
	if mid := outlines[0].Depths[outlines[0].Count/2]; mid > 100 {
		t.Errorf("the walker's reach ahead is %.1f, want it to stop on the hill", mid)
	}
	_, seen, _ = sceneIn(t, onHill, spawn{x: 0, y: 0, z: z(40, 2), sight: eyed(1)}, target)
	if seen[0].Count != 1 {
		t.Errorf("the hawk saw %d past the hill, want the target", seen[0].Count)
	}
}

func TestHeights_AFlatWorldRefusesAnEyeAndAQuasi3DWorldRefusesBlockers(t *testing.T) {
	expect := func(t *testing.T, want string, run func()) {
		t.Helper()
		defer func() {
			if msg, _ := recover().(string); !strings.Contains(msg, want) {
				t.Errorf("panic %q, want one mentioning %q", msg, want)
			}
		}()
		run()
		t.Errorf("no panic, want one mentioning %q", want)
	}
	expect(t, "Sight.Eye", func() { scene(t, spawn{x: 0, y: 0, sight: eyed(1.5)}) })
	blocked := &vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: math.Pi / 8, Radius: 300, Blockers: 1}
	expect(t, "Sight.Blockers", func() { sceneIn(t, &relief{}, spawn{x: 0, y: 0, sight: blocked}) })
}

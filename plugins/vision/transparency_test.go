package vision_test

import (
	"math"
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugins/vision"
)

// A forest 60 deep at τ = 0.5 costs 60 of reach on top of its depth, so the target 235 ahead
// costs 295: out of a 290 reach, in a 300 one.
func forestAhead(observer *vision.Sight) []spawn {
	return []spawn{
		{x: 0, y: 0, sight: observer, outline: true},
		{x: 100, y: 0, size: 60, tau: 0.5},
		{x: 240, y: 0},
	}
}

func TestScan_AVeilShortensTheReachWithoutBeingSeen(t *testing.T) {
	_, seen, outlines := scene(t, forestAhead(eastward(math.Pi/8, 290))...)
	if seen[0].Count != 0 {
		t.Errorf("saw %v through the forest at 290, want nothing", seen[0].IDs[:seen[0].Count])
	}
	mid := outlines[0].Depths[outlines[0].Count/2]
	if math.Abs(float64(mid)-230) > 2 { // 290 less the 60 the forest took
		t.Errorf("outline straight ahead reaches %.1f, want about 230", mid)
	}

	_, seen, _ = scene(t, forestAhead(eastward(math.Pi/8, 300))...)
	if seen[0].Count != 1 {
		t.Errorf("saw %d through the forest at 300, want the target alone", seen[0].Count)
	}
}

func TestScan_ClearSightLooksOverTheVeils(t *testing.T) {
	observer := eastward(math.Pi/8, 300)
	observer.Clear = true
	_, seen, outlines := scene(t, forestAhead(observer)...)
	if seen[0].Count != 1 {
		t.Errorf("saw %d over the forest, want the target alone", seen[0].Count)
	}
	if mid := outlines[0].Depths[outlines[0].Count/2]; math.Abs(float64(mid)-235) > 1 {
		t.Errorf("outline straight ahead reaches %.1f, want the target at 235", mid)
	}
}

func TestScan_WhatCutsSightCutsItForClearSightToo(t *testing.T) {
	for _, clear := range []bool{false, true} {
		observer := &vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: math.Pi / 8, Radius: 300, Clear: clear}
		_, seen, _ := scene(t,
			spawn{x: 0, y: 0, sight: observer},
			spawn{x: 100, y: 0, size: 60}, // tau 0: a wall
			spawn{x: 240, y: 0},
		)
		if seen[0].Count != 1 || seen[0].Dists[0] > 100 {
			t.Errorf("clear %v: saw %d at %v, want the wall alone", clear, seen[0].Count, seen[0].Dists[:seen[0].Count])
		}
	}
}

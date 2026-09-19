package world

import (
	"math"
	"testing"
)

// An entity straddling the world's corner breaks into four pieces, each clipped
// on both axes, so both axes have to pick their own slice of the sprite. The
// side and bottom pieces are where this used to go wrong: they were handed the
// whole sprite along the axis they had not wrapped on, smearing it across a
// piece only part as tall or as wide.
//
// The property that says it is right is that the pieces tile the texture: where
// the head stops, the tail starts, with no gap and no overlap.
func TestUVSpan_PiecesTileTheTextureAcrossTheCorner(t *testing.T) {
	// A 100x100 sprite at (970,960) of a 1000x1000 toroidal world splits into
	// 30x40 (main), 70x40 (right), 30x60 (bottom) and 70x60 (corner).
	const sprite, shifted = 100, -1000

	headU0, headU1 := uvSpan(30, sprite, 0)
	tailU0, tailU1 := uvSpan(70, sprite, shifted)
	headV0, headV1 := uvSpan(40, sprite, 0)
	tailV0, tailV1 := uvSpan(60, sprite, shifted)

	for _, tc := range []struct {
		axis           string
		h0, h1, t0, t1 float32
	}{
		{"across", headU0, headU1, tailU0, tailU1},
		{"down", headV0, headV1, tailV0, tailV1},
	} {
		if tc.h0 != 0 {
			t.Errorf("%s: the head starts at %v, want 0", tc.axis, tc.h0)
		}
		if tc.t1 != 1 {
			t.Errorf("%s: the tail ends at %v, want 1", tc.axis, tc.t1)
		}
		if math.Abs(float64(tc.h1-tc.t0)) > 1e-6 {
			t.Errorf("%s: the head ends at %v but the tail starts at %v — the pieces do not meet", tc.axis, tc.h1, tc.t0)
		}
	}

	// And the two axes really are decided apart: the right piece is a tail
	// across but still a head downwards.
	if _, u1 := uvSpan(70, sprite, shifted); u1 != 1 {
		t.Errorf("the right piece ends at %v across, want 1", u1)
	}
	if v0, _ := uvSpan(40, sprite, 0); v0 != 0 {
		t.Errorf("the right piece starts at %v downwards, want 0 — it never wrapped on that axis", v0)
	}
}

// A sprite that never reaches an edge shows all of itself.
func TestUVSpan_UnwrappedSpriteShowsWholeTexture(t *testing.T) {
	u0, u1 := uvSpan(100, 100, 0)
	if u0 != 0 || u1 != 1 {
		t.Errorf("u = [%v, %v], want the whole sprite [0, 1]", u0, u1)
	}
}

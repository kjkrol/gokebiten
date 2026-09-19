package vision

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// MaxSeen caps how many entities one scan records, nearest first — anything
// further is dropped, as collisions.Collision drops neighbours past
// MaxTouching. Sight is for reacting to what is closest, not for taking
// inventory.
const MaxSeen = 8

// Sizing of the outline buffer. The three together decide MaxSamples: an
// outline sampled at a fixed angular step drifts by Radius*2*HalfAngle/samples
// at full range, so widening the cone or reaching further needs more samples to
// hold the same accuracy.
const (
	// MaxSightRadius is the longest range the outline is sized for.
	MaxSightRadius = 300
	// MaxHalfAngleMilli is the widest half-angle the sizing assumes, in
	// milliradians — pi/6 is about 524.
	MaxHalfAngleMilli = 524
	// EdgeTolerance is how far a shadow edge may land from its true angle at
	// full range, in world units.
	EdgeTolerance = 5

	// MaxSamples follows from the three above — 64 as they stand, meaning
	// "accurate to EdgeTolerance out to MaxSightRadius for a cone no wider
	// than MaxHalfAngleMilli". A wider or longer cone still works; its outline
	// is simply coarser.
	//
	// Samples are one more than the gaps between them, and the gaps round up:
	// truncating would leave the widest cone drifting just past EdgeTolerance.
	// The arithmetic stays integral because converting a non-integral float
	// constant to int is a compile error in Go.
	arcMilli   = 2 * MaxHalfAngleMilli * MaxSightRadius
	gapMilli   = 1000 * EdgeTolerance
	MaxSamples = (arcMilli+gapMilli-1)/gapMilli + 1
)

// Sight is what an entity can take in: where it looks, how wide, how far.
// Facing is its own, independent of which way the entity is moving, so a
// sentry can stand still and look around while something else flies sideways
// watching where it came from.
type Sight struct {
	Facing    geom.Vec // unit vector
	HalfAngle float64  // radians either side of Facing
	Radius    float64  // world units
}

// Sighted is what the last scan found, nearest first — the seam every Behavior
// reads. Count says how many of the arrays are in use.
type Sighted struct {
	IDs   [MaxSeen]uid.UID64
	Dists [MaxSeen]float32
	Count uint8
}

// SightOutline is the drawn shape of one entity's view, held as a distance per
// evenly spaced angle across the cone — the angles follow from the index, so
// only the reach is stored.
//
// Its presence is what marks an entity for outline work. An entity without it
// is still scanned and still fills Sighted; it just costs nothing to draw and
// nothing to store.
type SightOutline struct {
	Depths [MaxSamples]float32
	Count  uint8
}

package vision

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/uid"
)

// MaxSeen caps how many entities one scan records, nearest first.
const MaxSeen = 8

// Sizing of the outline buffer: the first three decide MaxSamples.
const (
	// MaxSightRadius is the longest range the outline is sized for.
	MaxSightRadius = 300
	// MaxHalfAngleMilli is the widest half-angle the sizing assumes, in milliradians (pi/6).
	MaxHalfAngleMilli = 524
	// EdgeTolerance is how far a shadow edge may land from its true angle at full range.
	EdgeTolerance = 5

	// MaxSamples keeps a cone within the three limits above accurate to EdgeTolerance.
	arcMilli   = 2 * MaxHalfAngleMilli * MaxSightRadius
	gapMilli   = 1000 * EdgeTolerance
	MaxSamples = (arcMilli+gapMilli-1)/gapMilli + 1
)

// Sight is what an entity can take in — where it looks, how wide, how far — and
// what the last scan found there. Facing is its own, whichever way the entity moves.
type Sight struct {
	Facing    geom.Vec // unit vector
	HalfAngle float64  // radians either side of Facing
	Radius    float64  // world units
	Seen      Sighted  // nearest first
}

// Sighted is what one scan found, nearest first. Count says how many of the
// arrays are in use.
type Sighted struct {
	IDs   [MaxSeen]uid.UID64
	Dists [MaxSeen]float32
	Count uint8
}

// SightOutline is the drawn shape of one entity's view: a reach per evenly spaced angle
// across the cone. Only an entity carrying it has its outline computed.
type SightOutline struct {
	Depths [MaxSamples]float32
	Count  uint8
}

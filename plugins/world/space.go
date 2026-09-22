package world

import (
	"fmt"
	"log"
	"math"
	"math/bits"

	"github.com/kjkrol/aabbworld"
)

func buildSpace(cfg Config) *aabbworld.Space {
	const minCapacity, maxCapacity = 2.0, 8.0

	worldArea := uint64(cfg.Space.Width) * uint64(cfg.Space.Height)
	entityArea := uint64(cfg.Entities.MaxSize) * uint64(cfg.Entities.MaxSize)
	density := float64(uint64(cfg.Entities.MaxCount)*entityArea) / float64(worldArea)

	raw := math.Round(1.0 / math.Sqrt(density))
	capacity := uint32(math.Max(minCapacity, math.Min(maxCapacity, raw)))
	bucketSide := uint32(1) << bits.Len32(cfg.Entities.MaxSize*capacity-1)

	log.Printf("[world] maxEntities=%d, density=%.2f%%, capacity=%d → bucket=%dx%d, bucketCap=%d",
		cfg.Entities.MaxCount, density*100, capacity,
		bucketSide, bucketSide,
		capacity*capacity)

	space, err := aabbworld.NewSpace(aabbworld.Config{
		Width:          cfg.Space.Width,
		Height:         cfg.Space.Height,
		Edges:          cfg.Space.Edges,
		BucketSize:     bucketSide,
		BucketCapacity: int(capacity * capacity),
	})
	if err != nil {
		panic(fmt.Sprintf("world: invalid space configuration: %v", err))
	}
	return space
}

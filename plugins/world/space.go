package world

import (
	"fmt"
	"log"
	"math"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/spatial"
)

func buildSpace(cfg Config) *gokg.Space {
	const minCapacity, maxCapacity = 2.0, 8.0

	worldArea := uint64(cfg.Space.Width) * uint64(cfg.Space.Height)
	entityArea := uint64(cfg.Entities.MaxSize) * uint64(cfg.Entities.MaxSize)
	density := float64(uint64(cfg.Entities.MaxCount)*entityArea) / float64(worldArea)

	raw := math.Round(1.0 / math.Sqrt(density))
	capacity := uint32(math.Max(minCapacity, math.Min(maxCapacity, raw)))
	bucketResolution := spatial.ResolutionFrom(cfg.Entities.MaxSize * capacity)

	log.Printf("[world] maxEntities=%d, density=%.2f%%, capacity=%d → bucket=%dx%d, bucketCap=%d, opsBuffer=%d",
		cfg.Entities.MaxCount, density*100, capacity,
		bucketResolution.Side(), bucketResolution.Side(),
		capacity*capacity, cfg.Entities.MaxCount*8)

	space, err := gokg.NewSpace(gokg.Config{
		Width:          cfg.Space.Width,
		Height:         cfg.Space.Height,
		Toroidal:       cfg.Space.Toroidal,
		BucketSize:     bucketResolution,
		BucketCapacity: int(capacity * capacity),
		OpsBufferSize:  cfg.Entities.MaxCount * 8,
	})
	if err != nil {
		panic(fmt.Sprintf("world: invalid space configuration: %v", err))
	}
	return space
}

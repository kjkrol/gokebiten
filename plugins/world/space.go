package world

import (
	"fmt"
	"log"
	"math"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/spatial"
)

// Space returns world's shared spatial index — every Populate entity is kept in sync with it.
func (w *Module) Space() *gokg.Space { return w.space }

func buildSpace(cfg Config, pop Population) *gokg.Space {
	const minCapacity, maxCapacity = 2.0, 8.0

	worldArea := uint64(cfg.Width) * uint64(cfg.Height)
	entityArea := uint64(pop.MaxSize) * uint64(pop.MaxSize)
	density := float64(uint64(pop.MaxCount)*entityArea) / float64(worldArea)

	raw := math.Round(1.0 / math.Sqrt(density))
	capacity := uint32(math.Max(minCapacity, math.Min(maxCapacity, raw)))
	bucketResolution := spatial.ResolutionFrom(pop.MaxSize * capacity)

	log.Printf("[world] maxEntities=%d, density=%.2f%%, capacity=%d → bucket=%dx%d, bucketCap=%d, opsBuffer=%d",
		pop.MaxCount, density*100, capacity,
		bucketResolution.Side(), bucketResolution.Side(),
		capacity*capacity, pop.MaxCount*8)

	space, err := gokg.NewSpace(gokg.Config{
		Width:          cfg.Width,
		Height:         cfg.Height,
		Toroidal:       cfg.Toroidal,
		BucketSize:     bucketResolution,
		BucketCapacity: int(capacity * capacity),
		OpsBufferSize:  pop.MaxCount * 8,
	})
	if err != nil {
		panic(fmt.Sprintf("world: invalid space configuration: %v", err))
	}
	return space
}

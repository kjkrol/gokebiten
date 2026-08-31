package elastic

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

var _ collisions.CollisionHandler = (*Handler)(nil)

// Handler is a ready-made collision reaction: elastic velocity swap along
// the penetration axis. Independent of any bookkeeping — compose it with
// strategies/stats if you also want hit tracking.
type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) OnCollision(_ *goke.CmdBuf, e collisions.CollisionEvent) {
	if e.VelA == nil || e.VelB == nil {
		return // one side is immovable — nothing to swap
	}
	swapVelocity(e.VelA, e.VelB, e.Penetration)
}

// swapVelocity swaps the velocity components along the penetration axis,
// unless the objects are already separating on that axis.
func swapVelocity(velA, velB *world.Velocity, pen geom.Vec[int32]) {
	da, db := velA.Delta(), velB.Delta()
	switch {
	case pen.X != 0:
		relVelX := da.X - db.X
		if (pen.X > 0 && relVelX > 0) || (pen.X < 0 && relVelX < 0) {
			return
		}
		da.X, db.X = db.X, da.X
	case pen.Y != 0:
		relVelY := da.Y - db.Y
		if (pen.Y > 0 && relVelY > 0) || (pen.Y < 0 && relVelY < 0) {
			return
		}
		da.Y, db.Y = db.Y, da.Y
	default:
		return
	}
	velA.SetDelta(da)
	velB.SetDelta(db)
}

package world

import "github.com/kjkrol/uid"

// Moving is what an Each behavior hosted by the VelocitySystem gets, every tick, for every entity:
// scale Base.Vel.Value to slow or stop it, after Steering has written the base speed.
type Moving struct {
	ID   uid.UID64
	Base *Base
}

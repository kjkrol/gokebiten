package world

import (
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/uid"
)

// exits remembers who has left by an open edge and not come back, so each leaver is handled once.
type exits struct {
	outside map[uid.UID64]struct{}
	onExit  func(plugin.Tick, uid.UID64)
}

// quiet reports that nobody is outside, so an entity still inside needs no bookkeeping.
func (e *exits) quiet() bool { return len(e.outside) == 0 }

// left takes whether id is still inside and reports a leaver heard of for the first time.
func (e *exits) left(id uid.UID64, inside bool) bool {
	if inside {
		delete(e.outside, id)
		return false
	}
	if _, known := e.outside[id]; known {
		return false
	}
	if e.outside == nil {
		e.outside = make(map[uid.UID64]struct{})
	}
	e.outside[id] = struct{}{}
	return true
}

func (e *exits) forget(id uid.UID64) { delete(e.outside, id) }

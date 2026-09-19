package world

import "github.com/kjkrol/goke/v3"

// Behavior is one decision system, contributed by a plugin or by the game and
// run in the pass before movement — see Plugin.RegisterBehavior.
//
// It is a plain goke.System: build the query in Init, act in Update. Give that
// query an Include for the tag the behavior applies to, so it visits only the
// chunks holding those entities instead of every entity in the world.
type Behavior = goke.System

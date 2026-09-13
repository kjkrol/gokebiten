package world

// Entry is one entity to spawn: the EntKind named Kind, with Data feeding that kind's Load templates.
type Entry struct {
	Kind string
	Data any
}

// Roster lists entities for Plugin.Seed.
type Roster []Entry

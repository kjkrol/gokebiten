package world

// Entry is one entity to spawn — built only by EntKindDict.Entry, passed to Plugin.Seed.
type Entry struct {
	kind string
	data any
}

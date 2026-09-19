package world

// TypeID identifies an entity's kind at runtime. EntKindDict.Define assigns one
// per kind in call order; Resources.TypeNames maps it back to the kind's name.
type TypeID uint8

// MaxEntKinds is how many kinds a single dictionary can name, bounded by TypeID.
const MaxEntKinds = 1 << 8

// Type is the kind an entity was spawned from — the one trace of its EntKind
// that outlives spawning, and the only way a plugin can ask what an entity is.
// Every entity world spawns carries one.
type Type struct {
	ID TypeID
}

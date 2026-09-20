package world

// TypeID identifies an entity's kind at runtime, carried in every entity's Base
// — the one trace of its EntKind that outlives spawning. EntKindDict.Define
// assigns one per kind in call order; Resources.TypeNames maps it back to a name.
type TypeID uint8

// MaxEntKinds is how many kinds a single dictionary can name, bounded by TypeID.
const MaxEntKinds = 1 << 8

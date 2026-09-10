package resources

// Serializable is implemented by a Resources value (or tracked Plugin)
// contributing pointers for Persistence.Save/Load to gob-encode/decode.
type Serializable interface {
	Persisted() []any
}

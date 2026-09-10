package resources

// Resources marks a type as a plugin's published state, readable by other
// plugins/the game — never a Plugin itself or other behavior-bearing type.
type Resources interface {
	Resources()
}

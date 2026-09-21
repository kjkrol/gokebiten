package plugin

// Restorer is a tracked Plugin called back right after Persistence.Load has decoded its state.
// For a hook that queries loaded entities, see PostLoader.
type Restorer interface {
	Restore()
}

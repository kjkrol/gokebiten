package plugin

// Populator is implemented by a Plugin that seeds its own initial state —
// run only when a Stage starts without a restored save.
type Populator interface {
	Populate() error
}

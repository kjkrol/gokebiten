package plugin

// Restorer is implemented by a tracked Plugin needing a synchronous
// callback right after Persistence.Load decodes its Serializable's
// Persisted() pointers — for state with no dependency on ECS entities.
// See PostLoader for a hook that needs to query loaded entities instead.
type Restorer interface {
	Restore()
}

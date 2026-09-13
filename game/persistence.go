package game

// Persistence provides Save/Load/List for one Engine's ECS and resources — see Runtime.Persistence.
type Persistence interface {
	// List returns every save found for basePath, "" (quicksave) first.
	List(basePath string) ([]string, error)

	// Save writes resources and the ECS snapshot to disk under basePath/label, auto-including every tracked Serializable's targets.
	Save(basePath, label string, resources ...any) error

	// Load restores a snapshot written by Save, leaving anything absent from it unchanged.
	Load(basePath, label string, resources ...any) error
}

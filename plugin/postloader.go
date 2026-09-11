package plugin

import "github.com/kjkrol/goke/v3"

// PostLoader is implemented by a tracked value needing a one-time system
// run right after Persistence.Load restores entities.
type PostLoader interface {
	PostLoad() goke.System
}

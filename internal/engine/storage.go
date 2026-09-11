package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/kjkrol/gokebiten/plugin"
)

// storage is a name-keyed registry of installed plugins' Serializable
// state. Its only job is to hand Persistence.Save/Load a name-matched set
// of persist targets, so a save survives plugins being added, removed, or
// reordered between game versions instead of silently misreading bytes.
type storage struct {
	mu    sync.Mutex
	items map[string]plugin.Serializable
}

func newStorage() *storage {
	return &storage{items: make(map[string]plugin.Serializable)}
}

// register adds v under name (a Plugin's Name()) — install-time only.
func (s *storage) register(name string, v plugin.Serializable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	verifyPersisted(name, v)
	s.items[name] = v
}

// verifyPersisted panics if v.Persisted() contains anything gob can't encode.
func verifyPersisted(name string, v plugin.Serializable) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, target := range v.Persisted() {
		if err := enc.Encode(target); err != nil {
			panic(fmt.Sprintf("gokebiten: %q (%T).Persisted() returned an unencodable value %T: %v", name, v, target, err))
		}
	}
}

// persisted returns each registered item's Persisted() targets, keyed by
// its registration name — Save/Load match entries by this name, not by
// position, so adding/removing a plugin never shifts anyone else's data.
func (s *storage) persisted() map[string][]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]any, len(s.items))
	for name, v := range s.items {
		out[name] = v.Persisted()
	}
	return out
}

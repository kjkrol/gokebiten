package gokebiten

import (
	"fmt"
	"reflect"
	"sync"
)

// Resources is a typed registry, one value per concrete type T, keyed by reflect.Type.
// Resolve what you need once in Plugin.Install; never on the Update/Draw hot path.
type Resources struct {
	mu    sync.RWMutex
	items map[reflect.Type]any
}

// NewResources returns an empty resource registry.
func NewResources() *Resources {
	return &Resources{items: make(map[reflect.Type]any)}
}

// InsertResource registers v under its exact static type T, replacing whatever was there before.
func (r *Resources) InsertResource[T any](v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[reflect.TypeFor[T]()] = v
}

// GetResource returns the T resource, panicking if none is registered.
func (r *Resources) GetResource[T any]() T {
	v, ok := r.TryGetResource[T]()
	if !ok {
		var zero T
		panic(fmt.Sprintf("gokebiten: resource %T not registered", zero))
	}
	return v
}

// TryGetResource returns the T resource and whether it was registered.
func (r *Resources) TryGetResource[T any]() (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[reflect.TypeFor[T]()]
	if !ok {
		var zero T
		return zero, false
	}
	return v.(T), true
}

// RemoveResource deletes the T resource, if any.
func (r *Resources) RemoveResource[T any]() {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, reflect.TypeFor[T]())
}

// Resettable resources get Reset called each stats interval — see Game.Update.
type Resettable interface{ Reset() }

// forEach calls fn with every registered resource value.
func (r *Resources) forEach(fn func(any)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.items {
		fn(v)
	}
}

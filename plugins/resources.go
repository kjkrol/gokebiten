package plugins

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/kjkrol/gokebiten/plugins/resource"
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

// Get returns the T resource, panicking if none is registered.
func (r *Resources) Get[T resource.PluginResource]() T {
	v, ok := r.TryGet[T]()
	if !ok {
		var zero T
		panic(fmt.Sprintf("gokebiten: resource %T not registered", zero))
	}
	return v
}

// TryGet returns the T resource and whether it was registered.
func (r *Resources) TryGet[T resource.PluginResource]() (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[reflect.TypeFor[T]()]
	if !ok {
		var zero T
		return zero, false
	}
	return v.(T), true
}

// ForEach calls fn with every registered resource value.
func (r *Resources) ForEach(fn func(any)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.items {
		fn(v)
	}
}

// Insert registers v under its exact static type T, replacing whatever was there before.
func (r *Resources) Insert[T resource.PluginResource](v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[reflect.TypeFor[T]()] = v
}

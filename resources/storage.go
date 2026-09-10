package resources

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

// Storage is a typed registry, one value per concrete type T, keyed by reflect.Type.
// Resolve what you need once in Plugin.Install; never on the Update/Draw hot path.
type Storage struct {
	mu    sync.RWMutex
	items map[reflect.Type]Resources
}

// NewStorage builds an empty Storage registry.
func NewStorage() *Storage {
	return &Storage{items: make(map[reflect.Type]Resources)}
}

func (s *Storage) Get[T Resources]() T {
	v, ok := s.TryGet[T]()
	if !ok {
		var zero T
		panic(fmt.Sprintf("gokebiten: resource %T not registered", zero))
	}
	return v
}

func (s *Storage) TryGet[T Resources]() (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[reflect.TypeFor[T]()]
	if !ok {
		var zero T
		return zero, false
	}
	return v.(T), true
}

func (s *Storage) ForEach(fn func(Resources)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.items {
		fn(v)
	}
}

func (s *Storage) Insert[T Resources](v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sv, ok := any(v).(Serializable); ok {
		verifyPersisted(sv)
	}
	s.items[reflect.TypeFor[T]()] = v
}

// InsertDynamic registers v under its runtime concrete type — install-time only.
func (s *Storage) InsertDynamic(v Resources) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sv, ok := v.(Serializable); ok {
		verifyPersisted(sv)
	}
	s.items[reflect.TypeOf(v)] = v
}

// verifyPersisted panics if v.Persisted() contains anything gob can't encode.
func verifyPersisted(v Serializable) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, target := range v.Persisted() {
		if err := enc.Encode(target); err != nil {
			panic(fmt.Sprintf("gokebiten: %T.Persisted() returned an unencodable value %T: %v", v, target, err))
		}
	}
}

// Persisted aggregates Persisted() from every Serializable item, sorted by type name for deterministic ordering.
func (s *Storage) Persisted() []any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]reflect.Type, 0, len(s.items))
	for t := range s.items {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool { return types[i].String() < types[j].String() })

	var out []any
	for _, t := range types {
		if sv, ok := s.items[t].(Serializable); ok {
			out = append(out, sv.Persisted()...)
		}
	}
	return out
}

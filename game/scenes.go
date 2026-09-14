package game

import "fmt"

// Scenes is the immutable registry of every Scene a Stage can show, keyed by Name().
type Scenes interface {
	// Get resolves name to the Scene registered under it.
	Get(name string) (Scene, bool)

	// All returns every registered Scene.
	All() []Scene

	// Composition reports which Scenes are visible, in what order, and which is active.
	Composition() Composition
}

type stack struct {
	scenes      map[string]Scene
	all         []Scene
	composition Composition
}

var _ Scenes = (*stack)(nil)

// NewStack builds a Stack from scenes, erroring if two share a Name().
func NewStack(scenes ...Scene) (Scenes, error) {
	s := &stack{scenes: make(map[string]Scene, len(scenes)), all: scenes}
	for _, sc := range scenes {
		if _, exists := s.scenes[sc.Name()]; exists {
			return nil, fmt.Errorf("game: duplicate scene name %q", sc.Name())
		}
		s.scenes[sc.Name()] = sc
	}
	s.composition = newComposition(s)
	return s, nil
}

func (s *stack) Composition() Composition { return s.composition }

func (s *stack) Get(name string) (Scene, bool) {
	sc, ok := s.scenes[name]
	return sc, ok
}

func (s *stack) All() []Scene { return s.all }

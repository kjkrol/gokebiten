package game

import (
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/render"
)

// Scene groups the renderers and input handling for one thing a Stage can show.
type Scene interface {
	// Name uniquely identifies this scene.
	Name() string

	// Layers returns this scene's renderers, bottom to top; called once, when the Stage is entered.
	Layers() []render.Renderer

	// HandleEvents handles this tick's input, while this scene is active.
	HandleEvents(events *control.InputEvents, runtime Runtime, composition Composition)

	// Focusable reports whether this scene can ever become active.
	Focusable() bool
}

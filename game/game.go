package game

// Game is implemented by the user — Engine drives it.
type Game interface {
	// Props configures the window, tick rate, and world for the engine.
	Props() Props

	// Stages returns every Stage by Name(), and which one starts active.
	Stages() (stages map[string]Stage, initial string)
}

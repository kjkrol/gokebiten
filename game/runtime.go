package game

import "github.com/kjkrol/gokebiten/camera"

// TPS is the built-in measured-ticks-per-second counter.
type TPS struct{ Ticks int }

// Runtime is what a Game may keep from Init for later use (e.g. in
// HandleEvents) — pause control, Persistence, the measured TPS counter,
// the shared Camera, and Quit.
type Runtime interface {
	Paused() bool
	Pause()
	Resume()
	TogglePause()
	Persistence() Persistence
	TPS() *TPS
	Camera() camera.Camera
	Quit()
}

package game

// TPS is the built-in measured-ticks-per-second counter.
type TPS struct{ Ticks int }

// Runtime is what a Game may keep from Init for later use (e.g. in
// HandleEvents) — pause control, Persistence, and the measured TPS counter.
type Runtime interface {
	Paused() bool
	Pause()
	Resume()
	TogglePause()
	Persistence() Persistence
	TPS() *TPS
}

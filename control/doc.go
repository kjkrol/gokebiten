// Package control is the input vocabulary: what the engine captures each frame and what a
// handler is told.
//
// # InputEvents
//
// [InputEvents] is one tick's input: the mouse position and its delta, which modifiers and the
// middle button are down, whether the window fills the screen, the scroll delta, and a queue of
// [ClickEvent]s and [KeyEvent]s, each a key or button with a [KeyAction] (press or release).
// The engine fills it from the platform and clears the transient part (ResetTransient) after
// every tick.
//
// # EventHandler
//
// [EventHandler] reacts to this tick's input. The engine calls the active Scene's HandleEvents
// once per tick, after capture and before the Stage's Update; a plugin's own EventHandler, such
// as the world's camera controls or selection's click and drag, runs the same way.
package control

// Package camera is the view onto a world: screen conversion, culling, the visible window and its
// control. It is a root package, not a plugin; the world plugin builds one from its space and
// every renderer draws through it.
//
// # Camera
//
// A [Camera] converts between world and screen (ToScreen, FromScreen, ToScreenQuads, which splits
// a rectangle into the [Quad]s its wrapped images project to), culls (Visible, Bounds) and is
// controlled (MoveTo, Translate, ZoomIn, ZoomOut, with min and max zoom). It keeps its own window
// arithmetic: wrapping on a wrapping axis of the world, held inside the world on any other.
// [NewFromSpace] builds one over a width x height world with aabbworld edge rules, viewing all of
// it by default; [NewFromSpaceWithConfig] takes a [Config] with a viewport size and zoom limits.
//
// # State
//
// [State] is the persistable part — the viewport and zoom — which the Camera hands to saves
// through Persisted and takes back through Restore. Config is construction-time only.
//
// # FromScreenRect
//
// [FromScreenRect] converts a screen rectangle to world space, for click and drag logic; on a
// torus the result may need wrapping.
package camera

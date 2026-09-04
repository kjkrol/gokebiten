// Package camera builds and publishes the render.Camera every renderer
// draws through - sized from world.Config if a world.Plugin is present,
// else from the game's screen size - so registration order between world
// and camera never matters.
package camera

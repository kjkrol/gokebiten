package world

import "github.com/kjkrol/gram/render"

// Appearance is the sprite an entity is drawn from; the Renderer's behaviors may layer over it.
type Appearance struct {
	SpriteID render.SpriteID
}

package selection

// Selected marks an entity as currently selected by the player.
type Selected struct{}

// Selectable marks an entity the player may select; without it clicks and marquees pass it by.
type Selectable struct{}

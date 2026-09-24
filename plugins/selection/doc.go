// Package selection turns Select commands into a Selected tag on Selectable world entities;
// its default bindings make a left drag one (a click is a drag of no length, Shift adds).
// WithRenderer outlines what is selected.
//
// # Selectable, Selected and SelectionSystem
//
// [Tags] are two bits of selection's tag [Family], from [Plugin.Tags]: Selectable marks an
// entity the player may select — a unit, not a stretch of terrain; give it with kind.Tagged —
// and Selected one the player has selected, flipped in place. A [Select] names entities by id or
// by a box in world units, additive or not; the plugin is a plugin.Commander — [Plugin.Commands]
// is the inbox, [Plugin.DefaultBindings] a left drag through the player's camera — and the
// [SelectionSystem] drains the inbox into Selected tags.
//
// # Renderer
//
// [Plugin.WithRenderer] builds the [Renderer] outlining every Selected entity in a
// [HighlightStyle] ([DefaultHighlightStyle] is a thin red outline; [HighlightStyleFn] adapts a
// function); the marquee of a drag in progress is the players plugin's to draw.
package selection

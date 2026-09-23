// Package selection turns mouse input into a Selected tag on Selectable world entities:
// left-click to select, drag for a marquee, shift to add. WithRenderer outlines what is selected.
//
// # Selectable, Selected and SelectionSystem
//
// [Tags] are two bits of selection's tag [Family], from [Plugin.Tags]: Selectable marks an
// entity the player may select — a unit, not a stretch of terrain; give it with kind.Tagged —
// and Selected one the player has selected, flipped in place. The [DefaultEventHandler] turns left-click
// and left-drag into [Resources] — the drag in progress and a [PendingSelect] once the gesture
// completes — and the [SelectionSystem] resolves it against the world's space through the camera
// into Selected tags, additive with shift.
//
// # Renderer
//
// [Plugin.WithRenderer] builds the [Renderer] outlining every Selected entity and drawing the
// marquee of a drag in progress, in a [HighlightStyle] ([DefaultHighlightStyle] is a thin red
// outline; [HighlightStyleFn] adapts a function).
package selection

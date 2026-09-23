// Package effects puts temporary changes on entities: a tag granted for a while, a component
// altered and later restored. An effect is defined once, from a Spec of traits, and cast from
// anywhere — a behavior, a key, another plugin.
//
// # Spec, Grant and Alter
//
// [Plugin.Define] registers an effect from a [Spec]: [Lasts] is how long a cast holds (without it,
// until [Dispel]); [Stacking] lets casts pile up instead of refreshing; [Grant] gives tags of a
// family while the effect runs and takes them back after, unless another running effect grants
// them; [Alter] changes a component and restores the original after — several Alters of one
// component stack from the original in slot order, whichever ends first. The plugin keeps the
// originals itself and saves them with the game.
//
// # Active, Cast and Dispel
//
// [Active] is what an entity is under: up to [MaxEffects] slots, saved with it. [Plugin.Cast] and
// [Plugin.CastFor] put an effect on an entity — attaching Active when it has none — and the
// change lands with the plugin's next pass; [Plugin.Dispel] ends one early; [Plugin.Has] asks. [Plugin.OnIdle] tells every
// listener when an entity's last effect ended — the board built WithEffects lets a cell entity go. Effects host no behaviors of their own: they are what
// behaviors cast. Call [Plugin.RunPlan] after world's.
package effects

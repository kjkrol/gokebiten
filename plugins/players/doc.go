// Package players is whoever acts in the game: a [Player] with a camera, a View through it and a
// set of bindings, and the commands players issue — carried to the plugin that defined each one.
//
// # Commands and inboxes
//
// A command is an intention in the game's vocabulary, as data: selection.Select, navigation.MoveTo,
// [Pan] and [Zoom]. Whoever defines a command's type owns it: it calls [Plugin.Listen] once for
// the type and gets the [Inbox] the commands land in, and its own system drains the inbox in its
// own pass ([Inbox.Drain], every [Issued] with the player that gave it). [Plugin.Issue] is how a
// command comes in — from a binding, an AI, a network — and a type nobody listens for is
// [ErrUnknownCommand]. The plugins never ask who spoke.
//
// # Players and bindings
//
// [Plugin.Local] adds a player at this keyboard, looking through the world's camera. A [Binding]
// is a [Trigger] — [KeyPress], [ButtonPress], [Drag], [Wheel], [ButtonHeld], [CursorAtEdge], each
// with exactly its [Mods] — the command it issues, built by [Command] from a [Context] (the cursor,
// a drag's start and end, the wheel, and World or WorldBox through the player's camera), and a
// label saying what it does. Plugins ship their defaults (selection.DefaultBindings,
// navigation.Plugin.DefaultBindings, [CameraBindings]); a game binds them on a player with
// [Player.Bind], adds its own, or leaves some out. Two bindings on one trigger are refused at
// Bind; a binding whose command nobody listens for is refused when the Stage is set up.
// [Player.Bindings] is the list a help screen draws; [Player.DragBox] the drag in progress.
//
// # Order within a tick
//
// The active Scene hands the tick's input to [Plugin.EventHandler], which runs every local
// player's bindings and fills the inboxes; the plugins drain theirs in their RunPlan;
// [Plugin.RunPlan], called last, carries out Pan and Zoom on the issuing player's camera and
// empties whatever is left. Keys that are not a move in the game — pause, quit, a debug toggle —
// stay in the Scene's HandleEvents.
package players

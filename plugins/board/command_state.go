package board

// CommandState is board's live move-order input state — HandleEvents
// implementations write to it, CommandSystem reads/clears it. Published to
// Resources by Plugin, so a custom EventHandler can drive move orders
// without touching CommandSystem's internals.
type CommandState struct {
	PendingTarget *CellID
}

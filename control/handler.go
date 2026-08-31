package control

// EventHandler reacts to this tick's input events. Game calls HandleEvents
// once per tick, right after capture and before the user's Loop.
type EventHandler interface {
	HandleEvents(events *InputEvents)
}

// HandlerFn adapts a plain function to EventHandler.
type HandlerFn func(events *InputEvents)

func (f HandlerFn) HandleEvents(events *InputEvents) { f(events) }

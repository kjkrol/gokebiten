package engine

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
)

// DefaultController captures input each frame and, once an EventHandler is
// registered via SetHandler, runs it every tick.
type DefaultController struct {
	adapter InputAdapter
	events  *control.InputEvents
	handler control.EventHandler
}

func NewDefaultController(adapter InputAdapter, events *control.InputEvents) *DefaultController {
	return &DefaultController{adapter: adapter, events: events}
}

func (c *DefaultController) SetHandler(handler control.EventHandler) { c.handler = handler }

func (c *DefaultController) Capture(e *control.InputEvents) { c.adapter.Capture(e) }

func (c *DefaultController) Init(*goke.SysInit) {}

func (c *DefaultController) Update(_ *goke.CmdBuf, _ time.Duration) {
	if c.handler != nil {
		c.handler.HandleEvents(c.events)
	}
}

// HandlerFn adapts a plain function to control.EventHandler.
type HandlerFn func(events *control.InputEvents)

func (f HandlerFn) HandleEvents(events *control.InputEvents) { f(events) }

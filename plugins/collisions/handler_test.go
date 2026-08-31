package collisions_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
)

type recordingHandler struct {
	name  string
	order *[]string
	onFn  func(e collisions.CollisionEvent)
}

func (h recordingHandler) OnCollision(_ *goke.CmdBuf, e collisions.CollisionEvent) {
	*h.order = append(*h.order, h.name)
	if h.onFn != nil {
		h.onFn(e)
	}
}

type initTrackingHandler struct {
	recordingHandler
	initCalled bool
}

func (h *initTrackingHandler) Init(*goke.SysInit) { h.initCalled = true }

func TestMultiHandler_CallsInOrder(t *testing.T) {
	var order []string
	mh := collisions.MultiHandler(
		recordingHandler{name: "a", order: &order},
		recordingHandler{name: "b", order: &order},
		recordingHandler{name: "c", order: &order},
	)

	mh.OnCollision(nil, collisions.CollisionEvent{})

	want := []string{"a", "b", "c"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestMultiHandler_LaterHandlersSeeEarlierMutations(t *testing.T) {
	var order []string
	mh := collisions.MultiHandler(
		recordingHandler{name: "setter", order: &order, onFn: func(e collisions.CollisionEvent) {
			e.PosA.TopLeft.X = 1
		}},
		recordingHandler{name: "reader", order: &order, onFn: func(e collisions.CollisionEvent) {
			if e.PosA.TopLeft.X != 1 {
				t.Errorf("second handler should observe the first handler's mutation via the shared pointer, got TopLeft.X=%v", e.PosA.TopLeft.X)
			}
			e.PosA.TopLeft.X = 2
		}},
	)

	pos := &world.Position{}
	mh.OnCollision(nil, collisions.CollisionEvent{PosA: pos})

	if pos.TopLeft.X != 2 {
		t.Errorf("PosA.TopLeft.X = %v, want 2 (both handlers' mutations should land on the shared Position)", pos.TopLeft.X)
	}
}

func TestMultiHandler_Init_FansOutToInitializers(t *testing.T) {
	plain := recordingHandler{name: "plain", order: &[]string{}}
	initA := &initTrackingHandler{recordingHandler: recordingHandler{name: "a"}}
	initB := &initTrackingHandler{recordingHandler: recordingHandler{name: "b"}}

	mh := collisions.MultiHandler(plain, initA, initB)

	init, ok := mh.(collisions.Initializer)
	if !ok {
		t.Fatal("expected MultiHandler's result to implement collisions.Initializer")
	}
	init.Init(nil)

	if !initA.initCalled {
		t.Error("expected initA.Init to have been called")
	}
	if !initB.initCalled {
		t.Error("expected initB.Init to have been called")
	}
}

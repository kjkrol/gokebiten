package selection

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

func TestRenderer_Init_QueryMatchesOnlySelectedEntity(t *testing.T) {
	h := newHarness(t)
	selectedID := h.seed(50, 50, 10)
	unselectedID := h.seed(500, 500, 10)
	h.start()

	h.click(55, 55, false)
	if !h.isSelected(*selectedID) {
		t.Fatal("sanity check failed: expected the clicked entity to be selected")
	}
	if h.isSelected(*unselectedID) {
		t.Fatal("sanity check failed: expected the other entity to remain unselected")
	}

	r := NewRenderer(h.state)
	// RegSys calls Init immediately, so this builds r's query right away.
	h.ecs.RegSys(goke.SystemFn{OnInit: func(si *goke.SysInit) { r.Init(si) }})

	r.query.All()
	found := map[uid.UID64]bool{}
	for r.query.Next() {
		for _, id := range r.query.Cursor().IDs {
			found[id] = true
		}
	}
	if !found[*selectedID] {
		t.Error("expected Renderer's query to match the selected entity")
	}
	if found[*unselectedID] {
		t.Error("expected Renderer's query to NOT match the unselected entity")
	}
}

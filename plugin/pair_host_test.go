package plugin_test

import (
	"errors"
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/uid"
)

type body struct{ N int }

type hunter struct{ N int }

type hunted struct{ N int }

// sighting is a made-up host's description of a pair: who looks at whom.
type sighting struct{ from, to uid.UID64 }

// hostOf builds a host with behaviors registered, a hunter, a hunted, and their tag masks.
func hostOf(t *testing.T, behaviors ...plugin.Behavior) (h *plugin.PairHost[sighting], hunterMask, huntedMask uint64, pair sighting) {
	t.Helper()
	h = &plugin.PairHost[sighting]{}
	for _, b := range behaviors {
		if err := h.Add(b); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	goke.New().Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var bodies, hunters, prey goke.Comp[body]
		var asHunter goke.Comp[hunter]
		var asHunted goke.Comp[hunted]
		fh := si.NewFactory(&hunters, &asHunter)
		fh.Create(1)
		fh.Next()
		fp := si.NewFactory(&prey, &asHunted)
		fp.Create(1)
		fp.Next()
		pair = sighting{from: fh.IDs[0], to: fp.IDs[0]}

		var sought goke.Comp[body]
		walk, seek := si.NewQueryBuilder(&bodies), si.NewQueryBuilder(&sought)
		h.Bind(walk, seek)
		walking, seeking := walk.Build(), seek.Build()

		for walking.All(); walking.Next(); {
			for _, id := range walking.Cursor().IDs {
				if id == pair.from {
					hunterMask = h.InChunk(0, walking.Cursor())
				}
			}
		}
		if !seeking.Seek(pair.to) {
			t.Fatal("the hunted entity could not be sought")
		}
		huntedMask = h.At(1, seeking.Cursor())
	}})
	return h, hunterMask, huntedMask, pair
}

func recording(into *[]sighting) plugin.Behavior {
	return plugin.Between[hunter, hunted](func(_ plugin.Tick, s sighting) { *into = append(*into, s) })
}

func TestPairHost_Dispatch_KeepsTheDirection(t *testing.T) {
	var got []sighting
	h, hunterMask, huntedMask, pair := hostOf(t, recording(&got))

	h.Dispatch(plugin.Tick{}, hunterMask, huntedMask, pair)
	h.Dispatch(plugin.Tick{}, huntedMask, hunterMask, sighting{from: pair.to, to: pair.from})

	if len(got) != 1 || got[0] != pair {
		t.Errorf("behavior ran for %+v, want just the hunter looking at the hunted: %+v", got, pair)
	}
}

// A pair with no direction is handed over whichever way round the tags fit.
func TestPairHost_DispatchEitherWay_FindsTheFit(t *testing.T) {
	var got []sighting
	h, hunterMask, huntedMask, pair := hostOf(t, recording(&got))
	reversed := sighting{from: pair.to, to: pair.from}

	h.DispatchEitherWay(plugin.Tick{}, huntedMask, hunterMask, reversed, pair)

	if len(got) != 1 || got[0] != pair {
		t.Errorf("behavior ran for %+v, want it handed the pair with the hunter as Self: %+v", got, pair)
	}
}

func TestPairHost_Add_RefusesABehaviorMadeForAnotherHost(t *testing.T) {
	var h plugin.PairHost[sighting]
	stranger := plugin.Between[hunter, hunted](func(plugin.Tick, string) {})

	if err := h.Add(stranger); !errors.Is(err, plugin.ErrUnhostedBehavior) {
		t.Errorf("Add = %v, want ErrUnhostedBehavior", err)
	}
}

// Once the host's queries exist a behavior could no longer join them.
func TestPairHost_Add_RefusesOneThatComesAfterBind(t *testing.T) {
	var got []sighting
	h, _, _, _ := hostOf(t)

	if err := h.Add(recording(&got)); !errors.Is(err, plugin.ErrHostBuilt) {
		t.Errorf("Add after Bind = %v, want ErrHostBuilt", err)
	}
}

// group is a made-up host's description of one Self against many Others.
type group struct {
	others []int
	tags   []plugin.TagSet
}

func TestPairHost_DispatchGrouped_HandsOverTheOthersThatFit(t *testing.T) {
	var h plugin.PairHost[group]
	var got []group
	if err := h.Add(plugin.Between[hunter, hunted](func(_ plugin.Tick, g group) {
		got = append(got, group{others: append([]int(nil), g.others...)})
	})); err != nil {
		t.Fatalf("Add: %v", err)
	}
	const isHunter, isHunted = 1 << 0, 1 << 1
	build := func(matched []int) group { return group{others: matched} }

	h.DispatchGrouped(plugin.Tick{}, isHunter, []uint64{isHunted, 0, isHunted}, build)
	h.DispatchGrouped(plugin.Tick{}, isHunter, nil, build)
	h.DispatchGrouped(plugin.Tick{}, isHunted, []uint64{isHunted}, build)

	if len(got) != 2 {
		t.Fatalf("behavior ran %d times, want twice — once with prey in view, once with none", len(got))
	}
	if len(got[0].others) != 2 || got[0].others[0] != 0 || got[0].others[1] != 2 {
		t.Errorf("first group = %v, want others 0 and 2 — the one in between carries no tag", got[0].others)
	}
	if len(got[1].others) != 0 {
		t.Errorf("second group = %v, want it empty", got[1].others)
	}
}

func TestTagSet_Carries_AnswersForDeclaredTagsOnly(t *testing.T) {
	var h plugin.PairHost[group]
	if err := h.Add(plugin.Between[hunter, plugin.Anything](func(plugin.Tick, group) {}, plugin.Asking[hunted]())); err != nil {
		t.Fatalf("Add: %v", err)
	}
	const isHunted = 1 << 1

	if !h.TagSet(isHunted).Carries[hunted]() {
		t.Error("Carries[hunted] = false for an entity whose mask carries it")
	}
	if h.TagSet(0).Carries[hunted]() {
		t.Error("Carries[hunted] = true for an entity carrying nothing")
	}

	defer func() {
		if recover() == nil {
			t.Error("Carries of a tag nobody declared returned quietly, want a panic naming the fix")
		}
	}()
	h.TagSet(0).Carries[body]()
}

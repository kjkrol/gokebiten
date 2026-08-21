package physics_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics"
)

type expiryTag struct{ ExpiresNano int64 }

func expiryExtractor(tg *expiryTag) time.Time {
	if tg.ExpiresNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, tg.ExpiresNano)
}

// runTagExpiryScenario spawns one entity with expiryTag{ExpiresNano:
// expiresNano}, runs TagExpirySystem once, and reports whether the tag
// survived.
func runTagExpiryScenario(t *testing.T, expiresNano int64) (hasTag bool) {
	t.Helper()
	ecs := goke.New()
	var tag goke.Comp[expiryTag]
	var q *goke.Query

	ecs.Setup(
		goke.SystemFn{OnInit: func(si *goke.SysInit) {
			f := si.NewFactory(&tag)
			f.Create(1)
			f.Next()
			vals := tag.Slice(&f.Cursor)
			vals[0].ExpiresNano = expiresNano
		}},
		physics.NewTagExpirySystem(expiryExtractor),
		goke.SystemFn{OnInit: func(si *goke.SysInit) {
			q = si.NewQueryBuilder(&tag).Build()
		}},
	)

	q.All()
	for q.Next() {
		hasTag = true
	}
	return hasTag
}

func TestTagExpirySystem_ZeroNeverExpires(t *testing.T) {
	if !runTagExpiryScenario(t, 0) {
		t.Error("expected a zero (not-yet-initialized) expiresAt to never expire the tag")
	}
}

func TestTagExpirySystem_PastExpiry_RemovesTag(t *testing.T) {
	past := time.Now().Add(-time.Hour).UnixNano()
	if runTagExpiryScenario(t, past) {
		t.Error("expected a past expiresAt to have removed the tag")
	}
}

func TestTagExpirySystem_FutureExpiry_KeepsTag(t *testing.T) {
	future := time.Now().Add(time.Hour).UnixNano()
	if !runTagExpiryScenario(t, future) {
		t.Error("expected a future expiresAt to keep the tag")
	}
}

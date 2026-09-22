package main

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/game"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/world"
)

// benchInit is a game.Initializer that drives the real Stage without a window;
// Scene.Layers() is left out.
type benchInit struct {
	ecs     *goke.ECS
	world   *world.Plugin
	tracked []any
	pending []func() []goke.System
	tps     game.TPS
}

var _ game.Initializer = (*benchInit)(nil)

func (c *benchInit) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(*goke.SysInit) { m.RegSystems(c.ecs) }}
	c.tracked = append(c.tracked, m)
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}

func (c *benchInit) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.tracked = append(c.tracked, p)
		c.pending = append(c.pending, p.SetupSystems)
	}
}

func (c *benchInit) RegSys(factory func() goke.System) goke.Runnable { return c.ecs.RegSys(factory()) }
func (c *benchInit) ECS() *goke.ECS                                  { return c.ecs }
func (c *benchInit) TPS() *game.TPS                                  { return &c.tps }

func (c *benchInit) Use(p plugin.Plugin) error {
	c.tracked = append(c.tracked, p)
	return p.Install(c)
}

func (c *benchInit) Track(s plugin.Serializable) error {
	c.tracked = append(c.tracked, s)
	return nil
}

func (c *benchInit) UseWorld(cfg world.Config) *world.Plugin {
	cfg.Camera.ViewportWidth = ScreenWidth
	cfg.Camera.ViewportHeight = ScreenHeight
	c.world = world.NewPlugin(cfg)
	c.tracked = append(c.tracked, c.world)
	if err := c.world.Install(c); err != nil {
		panic(err)
	}
	return c.world
}

// buildStage runs the fresh-spawn half of entering a Stage: Init, Spawn, Populate, Setup.
func buildStage(tb testing.TB) (*goke.ECS, *mainStage) {
	tb.Helper()

	rng = rand.New(rand.NewPCG(0x5eed, 0xc0ffee))

	stage := &mainStage{}
	ctx := &benchInit{ecs: goke.New()}
	if err := stage.Init(ctx); err != nil {
		tb.Fatalf("Init: %v", err)
	}
	if err := stage.Spawn(); err != nil {
		tb.Fatalf("Spawn: %v", err)
	}
	for _, v := range ctx.tracked {
		if p, ok := v.(plugin.Populator); ok {
			if err := p.Populate(); err != nil {
				tb.Fatalf("Populate: %v", err)
			}
		}
	}
	ctx.ecs.SetPlan(stage.Update)

	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)

	return ctx.ecs, stage
}

// withScale runs fn with the demo sized to rect/percent, restoring the defaults afterwards.
func withScale(rect uint32, percent float64, fn func()) {
	oldRect, oldFill, oldCount := RectSize, FillPercent, EntityCount
	RectSize, FillPercent, EntityCount = rect, percent, countFor(rect, percent)
	defer func() { RectSize, FillPercent, EntityCount = oldRect, oldFill, oldCount }()
	fn()
}

// benchScales are the two configurations the slowdown report is about: the
// demo's own defaults, and the one where TPS collapses.
var benchScales = []struct {
	name    string
	rect    uint32
	percent float64
}{
	{"rect=20,fill=20%", 20, 20},
	{"rect=10,fill=20%", 10, 20},
	{"rect=20,fill=40%", 20, 40},
	{"rect=10,fill=40%", 10, 40},
	{"rect=8,fill=20%", 8, 20},
	{"rect=5,fill=20%", 5, 20},
}

const benchStep = time.Second / TPS

func BenchmarkStageTick(b *testing.B) {
	for _, sc := range benchScales {
		b.Run(sc.name, func(b *testing.B) {
			withScale(sc.rect, sc.percent, func() {
				ecs, stage := buildStage(b)
				for range 120 {
					ecs.Tick(benchStep)
				}
				b.ReportMetric(float64(EntityCount), "entities")
				b.ResetTimer()
				for b.Loop() {
					ecs.Tick(benchStep)
				}
				b.StopTimer()
				_ = stage
			})
		})
	}
}

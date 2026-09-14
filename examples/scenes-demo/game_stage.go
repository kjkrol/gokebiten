package main

import (
	"fmt"
	"image/color"
	"log"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

const (
	EntityCount = 12
	EntitySize  = 16

	saveBasePath = "scenes-demo"
	moverKind    = "mover"
)

// =========================== Stage ===========================

// GameplayStage is the real game — its own fresh ECS, built only once entered from the menu.
type GameplayStage struct {
	world *world.Plugin

	stack game.Stack
	panel *panelScene

	// SaveBasePath overrides where saves are read/written; tests set this to a temp path.
	SaveBasePath string
}

func (g *GameplayStage) basePath() string {
	if g.SaveBasePath != "" {
		return g.SaveBasePath
	}
	return saveBasePath
}

var _ game.Stage = (*GameplayStage)(nil)

func (g *GameplayStage) Name() string { return "gameplay" }

func (g *GameplayStage) Init(ctx game.Initializer) error {
	g.world = ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: EntityCount, MinSize: EntitySize, MaxSize: EntitySize},
	})
	velocity := world.Velocity{}
	velocity.SetDelta(geom.NewVec[int32](30, 20))
	g.world.EntKindDict().Define(func(k world.Kind[world.Position]) world.EntKind {
		return world.EntKind{
			Name:     moverKind,
			Position: k.Load(func(p world.Position) world.Position { return p }),
			Velocity: world.Const(velocity),
		}
	})

	worldScn := &worldScene{stage: g}
	g.panel = &panelScene{stage: g}
	hud := &hudScene{stage: g}

	stack, err := game.NewStack(worldScn, g.panel, hud)
	if err != nil {
		return err
	}
	g.stack = stack
	comp := stack.Composition()
	comp.Show(worldScn.Name())
	comp.Show(hud.Name())
	return ctx.Track(comp)
}

func (g *GameplayStage) Restore(p game.Persistence) (bool, error) {
	saves, err := p.List(g.basePath())
	if err != nil {
		return false, err
	}
	if !slices.Contains(saves, "") {
		return false, nil
	}
	if err := p.Load(g.basePath(), ""); err != nil {
		return false, err
	}
	return true, nil
}

func (g *GameplayStage) Spawn() error {
	placement := world.NewGridPlacement(ScreenWidth, ScreenHeight, EntitySize)
	kinds := g.world.EntKindDict()
	entries := make([]world.Entry, EntityCount)
	for i := range entries {
		entries[i] = kinds.Entry(moverKind, placement.Place(i, EntityCount))
	}
	g.world.Seed(entries...)
	return nil
}

func (g *GameplayStage) Update(ctx goke.RunCtx, d time.Duration) {
	g.world.RunPlan(ctx, d)
	ctx.Sync()
}

func (g *GameplayStage) Stack() game.Stack { return g.stack }

// handleGlobalKeys handles quit/pause/save — shared by worldScene and panelScene.
func handleGlobalKeys(events *control.InputEvents, runtime game.Runtime, basePath string) {
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeyEscape:
			runtime.Quit()
		case ebiten.KeySpace:
			runtime.TogglePause()
		case ebiten.KeyF5:
			if err := runtime.Persistence().Save(basePath, ""); err != nil {
				log.Printf("save: %v", err)
			} else {
				log.Print("saved (composition included: panel visibility survives Load)")
			}
		}
	}
}

// =========================== Scene ===========================

// worldScene draws the moving entities — always visible, and active
// whenever the panel isn't shown. P opens the panel.
type worldScene struct{ stage *GameplayStage }

var _ game.Scene = (*worldScene)(nil)

func (w *worldScene) Name() string { return "world" }

func (w *worldScene) Layers() []func() render.Renderer {
	s := w.stage

	kinds := s.world.EntKindDict()
	mover, _ := kinds.Get(moverKind)
	atlas := render.NewAtlas(EntitySize, len(kinds.All()))
	atlas.RegisterAt(mover.SpriteID, render.Solid(color.RGBA{R: 90, G: 200, B: 110, A: 255}))
	atlas.Close()
	s.world.WithRenderer(atlas)

	return []func() render.Renderer{
		func() render.Renderer {
			return render.NewCachedRenderer(render.SolidBackground{Color: color.RGBA{R: 30, G: 30, B: 40, A: 255}}, ScreenWidth, ScreenHeight)
		},
		s.world.Renderer,
	}
}

func (w *worldScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, composition game.Composition) {
	handleGlobalKeys(events, runtime, w.stage.basePath())
	for _, k := range events.KeyEvents {
		if k.Action == control.ActionPress && k.Key == ebiten.KeyP {
			composition.Show(w.stage.panel.Name())
		}
	}
}

func (w *worldScene) Focusable() bool { return true }

// panelScene is a modal box toggled by P: while shown, it sits on top of
// worldScene and becomes Composition.Active, so only its own HandleEvents
// runs — the world keeps ticking (GameplayStage.Update doesn't consult
// Composition at all), it just stops receiving input.
type panelScene struct{ stage *GameplayStage }

var _ game.Scene = (*panelScene)(nil)

func (p *panelScene) Name() string { return "panel" }

func (p *panelScene) Layers() []func() render.Renderer {
	return []func() render.Renderer{
		func() render.Renderer { return &panelRenderer{} },
	}
}

func (p *panelScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, composition game.Composition) {
	handleGlobalKeys(events, runtime, p.stage.basePath())
	for _, k := range events.KeyEvents {
		if k.Action == control.ActionPress && k.Key == ebiten.KeyP {
			composition.Hide(p.Name())
		}
	}
}

func (p *panelScene) Focusable() bool { return true }

type panelRenderer struct{}

func (r *panelRenderer) Init(*goke.SysInit) {}

func (r *panelRenderer) Draw(screen *ebiten.Image) {
	const w, h = 300, 140
	x, y := float32(ScreenWidth-w)/2, float32(ScreenHeight-h)/2
	vector.FillRect(screen, x, y, w, h, color.RGBA{R: 235, G: 235, B: 235, A: 255}, false)
	ebitenutil.DebugPrintAt(screen, "PANEL\n\nthe world keeps ticking behind me\nP to close", int(x)+12, int(y)+12)
}

// hudScene is a passive overlay — always shown on top (even over the
// panel), but Focusable() is false so it never becomes Composition.Active
// and never intercepts input, no matter what's drawn beneath it.
type hudScene struct{ stage *GameplayStage }

var _ game.Scene = (*hudScene)(nil)

func (h *hudScene) Name() string { return "hud" }

func (h *hudScene) Layers() []func() render.Renderer {
	return []func() render.Renderer{
		func() render.Renderer { return &hudRenderer{stage: h.stage} },
	}
}

func (h *hudScene) HandleEvents(*control.InputEvents, game.Runtime, game.Composition) {}

func (h *hudScene) Focusable() bool { return false }

type hudRenderer struct{ stage *GameplayStage }

func (r *hudRenderer) Init(*goke.SysInit) {}

func (r *hudRenderer) Draw(screen *ebiten.Image) {
	active := r.stage.stack.Composition().Active()
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("active scene: %s  (P: toggle panel, F5: save)", active), 8, ScreenHeight-20)
}

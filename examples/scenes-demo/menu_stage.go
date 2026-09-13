package main

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/render"
)

// =========================== Stage ===========================

// MenuStage is the splash/menu — no entities, no gameplay plugins.
type MenuStage struct {
	gameplayName string

	stack game.Stack
}

// NewMenuStage builds a MenuStage that switches to the Stage named gameplayName on start.
func NewMenuStage(gameplayName string) *MenuStage {
	return &MenuStage{gameplayName: gameplayName}
}

var _ game.Stage = (*MenuStage)(nil)

func (m *MenuStage) Name() string { return "menu" }

func (m *MenuStage) Init(ctx game.Initializer) error {
	main := &menuScene{gameplayName: m.gameplayName}
	stack, err := game.NewStack(main)
	if err != nil {
		return err
	}
	m.stack = stack
	m.Composition().Show(main.Name())
	return ctx.Track(m.Composition())
}

func (m *MenuStage) Restore(game.Persistence) (bool, error) { return false, nil }

func (m *MenuStage) Spawn() error { return nil }

func (m *MenuStage) Update(goke.RunCtx, time.Duration) {}

func (m *MenuStage) Stack() game.Stack             { return m.stack }
func (m *MenuStage) Composition() game.Composition { return m.stack.Composition() }

// =========================== Scene ===========================

// menuScene shows the splash text and requests the gameplay Stage on
// Enter. Since MenuStage has no HandleEvents of its own, this is also
// where Escape-to-quit lives.
type menuScene struct{ gameplayName string }

var _ game.Scene = (*menuScene)(nil)

func (m *menuScene) Name() string { return "menu" }

func (m *menuScene) Layers() []func() render.Renderer {
	return []func() render.Renderer{
		func() render.Renderer { return &menuRenderer{} },
	}
}

func (m *menuScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, composition game.Composition) {
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeyEnter:
			if err := runtime.SwitchStage(m.gameplayName); err != nil {
				log.Printf("switch stage: %v", err)
			}
		case ebiten.KeyEscape:
			runtime.Quit()
		}
	}
}

func (m *menuScene) Focusable() bool { return true }

type menuRenderer struct{}

func (r *menuRenderer) Init(*goke.SysInit) {}

func (r *menuRenderer) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 20, B: 30, A: 255})
	ebitenutil.DebugPrintAt(screen, "gokebiten Stage/Scene demo\n\nPress ENTER to start", 20, 20)
}

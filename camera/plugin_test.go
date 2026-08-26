package camera_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/world"
	"github.com/kjkrol/gokg/geom"
)

func TestPlugin_Install_PublishesRenderCameraResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{ScreenWidth: 800, ScreenHeight: 600})
	plugin := camera.NewPlugin()

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	got, ok := game.Resources().TryGetResource[render.Camera]()
	if !ok {
		t.Fatal("expected render.Camera to be registered as a resource")
	}
	if got != plugin.Camera() {
		t.Error("registered render.Camera resource is not plugin.Camera()")
	}
}

func TestPlugin_Install_KeyedByInterfaceNotConcrete(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{ScreenWidth: 800, ScreenHeight: 600})
	if err := game.UsePlugin(camera.NewPlugin()); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	if _, ok := game.Resources().TryGetResource[*render.BasicCamera](); ok {
		t.Error("expected *render.BasicCamera to NOT be registered under its concrete type — only render.Camera (the interface)")
	}
}

func TestPlugin_Install_DefaultViewportMatchesScreenSize(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{ScreenWidth: 800, ScreenHeight: 600})
	plugin := camera.NewPlugin()
	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	bounds := plugin.Camera().Bounds()
	if bounds.TopLeft.X != 0 || bounds.TopLeft.Y != 0 {
		t.Errorf("Bounds().TopLeft = %+v, want (0,0)", bounds.TopLeft)
	}
	if w := bounds.BottomRight.X - bounds.TopLeft.X; w != 800 {
		t.Errorf("Bounds() width = %d, want 800", w)
	}
	if h := bounds.BottomRight.Y - bounds.TopLeft.Y; h != 600 {
		t.Errorf("Bounds() height = %d, want 600", h)
	}
}

func TestPlugin_Install_WithViewportOverride(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{ScreenWidth: 800, ScreenHeight: 600})
	override := geom.NewAABBAt(geom.NewVec[uint32](100, 100), 50, 30)
	plugin := camera.NewPlugin().WithViewport(override)

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	got := plugin.Camera().Bounds()
	if got != override {
		t.Errorf("Bounds() = %+v, want overridden viewport %+v (not the screen-derived default)", got, override)
	}
}

func TestPlugin_Install_TogglesToroidalSurface(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{ScreenWidth: 20, ScreenHeight: 20})
	game.Resources().InsertResource(world.Config{Width: 20, Height: 20, Toroidal: true})

	plugin := camera.NewPlugin()
	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	plugin.Camera().MoveTo(2, 2)
	plugin.Camera().Translate(-10, 0)

	got := plugin.Camera().Bounds()
	if got.TopLeft.X != 12 {
		t.Errorf("after wrap-around Translate, Bounds().TopLeft.X = %d, want 12 (wrapped, not clamped)", got.TopLeft.X)
	}
}

func TestPlugin_Install_EuclideanFallbackWithoutWorldConfig(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{ScreenWidth: 20, ScreenHeight: 20})

	plugin := camera.NewPlugin()
	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	plugin.Camera().MoveTo(2, 2)
	plugin.Camera().Translate(-10, 0)

	got := plugin.Camera().Bounds()
	if got.TopLeft.X != 0 {
		t.Errorf("without world.Config, Bounds().TopLeft.X = %d, want 0 (clamped euclidean fallback)", got.TopLeft.X)
	}
}

func TestPlugin_Name(t *testing.T) {
	p := camera.NewPlugin()
	if p.Name() != "gokebiten.camera" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.camera")
	}
}

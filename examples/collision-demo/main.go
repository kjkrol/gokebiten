package main

import (
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func main() {
	gokebiten.NewEngine(&gokebiten.Props{
		Title:       "GOKe + GOKg + Ebiten Integration",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
		World: world.Config{
			Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
			Entities: world.EntitiesCfg{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
		},
	}, &Demo{}).Run()
}

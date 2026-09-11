package main

import (
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func main() {
	gokebiten.NewEngine(&gokebiten.Props{
		Title:       "gokebiten board & navigation plugins demo",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
		World: world.Config{
			Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: false},
			Entities: world.EntitiesCfg{MaxCount: MaxEntCount, MinSize: EntitySize, MaxSize: EntitySize},
		},
	}, &Demo{}).Run()
}

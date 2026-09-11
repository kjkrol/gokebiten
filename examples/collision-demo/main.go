package main

import "github.com/kjkrol/gokebiten"

func main() {
	gokebiten.NewEngine(&gokebiten.Props{
		Title:       "GOKe + GOKg + Ebiten Integration",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	}, &Demo{}).Run()
}

package main

import "github.com/kjkrol/gokebiten"

func main() {
	gokebiten.NewEngine(&gokebiten.Props{
		Title:       "gokebiten board & navigation plugins demo",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	}, &Demo{}).Run()
}

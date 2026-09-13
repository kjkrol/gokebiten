package main

import (
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/world"
)

const (
	ScreenWidth  = 640
	ScreenHeight = 480
	TPS          = 60
)

// =========================== Game ===========================

// Demo is the thinnest possible game.Game — just the two Stages and which
// one starts active. All real behavior lives on MenuStage/GameplayStage.
type Demo struct {
	menu     *MenuStage
	gameplay *GameplayStage
}

var _ game.Game = (*Demo)(nil)

func NewDemo() *Demo {
	gameplay := GameplayStage{}
	return &Demo{
		gameplay: &gameplay,
		menu:     NewMenuStage(gameplay.Name()),
	}
}

func (d *Demo) Props() game.Props {
	return game.Props{
		Title:       "gokebiten Stage/Scene demo",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
		World: world.Config{
			Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
			Entities: world.EntitiesCfg{MaxCount: EntityCount, MinSize: EntitySize, MaxSize: EntitySize},
		},
	}
}

func (d *Demo) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{
		d.menu.Name():     d.menu,
		d.gameplay.Name(): d.gameplay,
	}, d.menu.Name()
}

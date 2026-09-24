package players

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gram/control"
)

// translator turns one tick's input into commands: every local player's bindings, in order.
type translator struct{ p *Plugin }

var _ control.EventHandler = translator{}

func (t translator) HandleEvents(ev *control.InputEvents) {
	mods := Mods{Shift: ev.Modifiers.Shift, Ctrl: ev.Modifiers.Ctrl, Alt: ev.Modifiers.Alt}
	for _, pl := range t.p.locals {
		pl.cursor = ev.MousePos
		ctx := Context{Player: pl, Cursor: ev.MousePos, Delta: ev.CursorDelta, Wheel: ev.ScrollDelta,
			Screen: pl.screen(), Mods: mods, FillsScreen: ev.WindowFillsScreen}

		for _, k := range ev.KeyEvents {
			if k.Action == control.ActionPress {
				t.fire(pl, KeyPress{Key: k.Key, Mods: mods}, ctx)
			}
		}
		for _, c := range ev.ClickQueue {
			pl.cursor = c.Pos
			at := ctx
			at.Cursor = c.Pos
			switch c.Action {
			case control.ActionPress:
				pl.press(c.Button, c.Pos)
				at.Start = c.Pos
				t.fire(pl, ButtonPress{Button: c.Button, Mods: mods}, at)
			case control.ActionRelease:
				if start, down := pl.release(c.Button); down {
					at.Start = start
					t.fire(pl, Drag{Button: c.Button, Mods: mods}, at)
				}
			}
		}
		if ev.ScrollDelta != 0 {
			t.fire(pl, Wheel{}, ctx)
		}

		inside := ev.MousePos.X >= 0 && ev.MousePos.X < ctx.Screen.X && ev.MousePos.Y >= 0 && ev.MousePos.Y < ctx.Screen.Y
		if !inside {
			continue
		}
		if ev.CursorDelta.X != 0 || ev.CursorDelta.Y != 0 {
			for button := range pl.held {
				t.fire(pl, ButtonHeld{Button: button}, ctx)
			}
			if _, down := pl.held[ebiten.MouseButtonMiddle]; ev.MiddleDown && !down {
				t.fire(pl, ButtonHeld{Button: ebiten.MouseButtonMiddle}, ctx)
			}
		}
		if atEdge(ctx) {
			t.fire(pl, CursorAtEdge{}, ctx)
		}
	}
}

// fire issues the command of every binding of pl on trigger that builds one.
func (t translator) fire(pl *Player, trigger Trigger, ctx Context) {
	for _, b := range pl.bindings {
		if b.Trigger != trigger {
			continue
		}
		if cmd, ok := b.build(ctx); ok {
			if err := t.p.Issue(pl, cmd); err != nil {
				panic(err)
			}
		}
	}
}

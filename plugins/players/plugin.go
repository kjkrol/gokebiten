package players

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/render"
)

// ErrUnknownCommand is what Issue reports for a command type no plugin listens for.
var ErrUnknownCommand = errors.New("players: no plugin listens for this command")

// Plugin keeps the game's players and carries their commands: whatever a player's bindings, an
// AI or a network issue lands in the inbox of the plugin that defined the command's type.
type Plugin struct {
	worldPlugin *world.Plugin
	locals      []*Player
	inboxes     map[reflect.Type]mailbox
	pans        Inbox[Pan]
	zooms       Inbox[Zoom]
	module      *module
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds the players plugin over worldPlugin, whose camera and View the local players share.
func NewPlugin(worldPlugin *world.Plugin) *Plugin {
	p := &Plugin{worldPlugin: worldPlugin, inboxes: map[reflect.Type]mailbox{}}
	p.inboxes[reflect.TypeFor[Pan]()] = &p.pans
	p.inboxes[reflect.TypeFor[Zoom]()] = &p.zooms
	return p
}

// Local adds a player at this keyboard, looking through the world's camera; bind it before Use.
func (p *Plugin) Local(name string) *Player {
	pl := &Player{ID: ID(len(p.locals)), Name: name, Camera: p.worldPlugin.Camera(), View: p.worldPlugin.View()}
	p.locals = append(p.locals, pl)
	return pl
}

// Locals lists the players at this keyboard.
func (p *Plugin) Locals() []*Player { return p.locals }

// Listen makes the caller the one owner of commands of type C and returns the inbox they land in;
// call it in the plugin's constructor or Install. A second Listen for C panics.
func (p *Plugin) Listen[C any]() *Inbox[C] {
	t := reflect.TypeFor[C]()
	if _, taken := p.inboxes[t]; taken {
		panic(fmt.Sprintf("players: %v already has a listener", t))
	}
	in := &Inbox[C]{}
	p.inboxes[t] = in
	return in
}

// Issue gives cmd as player (nil for nobody in particular); ErrUnknownCommand when no plugin
// listens for its type. Bindings, an AI or a network all come in here.
func (p *Plugin) Issue(player *Player, cmd any) error {
	box, ok := p.inboxes[reflect.TypeOf(cmd)]
	if !ok {
		return fmt.Errorf("%w: %T", ErrUnknownCommand, cmd)
	}
	box.put(player, cmd)
	return nil
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gram.players" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	p.module = &module{p: p, camera: &cameraSystem{pans: &p.pans, zooms: &p.zooms}}
	ctx.UseModule(p.module)
	return nil
}

// RunPlan carries out the camera commands and empties every inbox; call it last in Update.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer is a no-op — players draw nothing of their own.
func (p *Plugin) WithRenderer(render.AtlasSource) {}

// Renderer returns nil — players draw nothing of their own.
func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler translates this tick's input through every local player's bindings; call it from
// the active Scene's HandleEvents.
func (p *Plugin) EventHandler() control.EventHandler { return translator{p} }

// Serializable returns nil — the local players' cameras are the world's, saved with it.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// RegisterBehavior reports ErrUnhostedBehavior — players host no behaviors; they carry commands.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	for _, b := range behaviors {
		return fmt.Errorf("%w: %T in %s", plugin.ErrUnhostedBehavior, b, p.Name())
	}
	return nil
}

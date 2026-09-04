package gokebiten

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins"
)

// pluginManager installs plugins (retrying until their dependencies are available) and tracks what they register.
type pluginManager struct {
	game       *Game
	plugins    map[string]plugins.Plugin
	pending    []plugins.Plugin
	waitingOn  map[string]reflect.Type
	registered []any
}

// install queues p, resolves as much of the pending queue as possible, and rejects a duplicate Name.
func (m *pluginManager) install(p plugins.Plugin) error {
	if m.plugins == nil {
		m.plugins = make(map[string]plugins.Plugin)
	}
	if _, dup := m.plugins[p.Name()]; dup {
		return fmt.Errorf("gokebiten: plugin %q already installed", p.Name())
	}
	m.plugins[p.Name()] = p
	m.pending = append(m.pending, p)
	return m.resolvePending()
}

// track records v so providedComps/postLoadSystems/saveTargets can find it later.
func (m *pluginManager) track(v any) { m.registered = append(m.registered, v) }

// providedComps collects LoadComps from every tracked value implementing goke.CompProvider.
func (m *pluginManager) providedComps() []goke.CompToken { return goke.ProvidedComps(m.registered...) }

// postLoadSystems collects PostLoad from every tracked value implementing PostLoader.
func (m *pluginManager) postLoadSystems() []goke.System {
	var systems []goke.System
	for _, v := range m.registered {
		if pl, ok := v.(PostLoader); ok {
			systems = append(systems, pl.PostLoad())
		}
	}
	return systems
}

// saveTargets collects SaveTargets from every tracked value implementing Saveable.
func (m *pluginManager) saveTargets() []any {
	var out []any
	for _, v := range m.registered {
		if s, ok := v.(Saveable); ok {
			out = append(out, s.SaveTargets()...)
		}
	}
	return out
}

// resolvePending installs every pending plugin whose dependencies are satisfied, retrying until no more progress.
func (m *pluginManager) resolvePending() error {
	for {
		progressed := false
		var stillPending []plugins.Plugin
		for _, p := range m.pending {
			ctx := plugins.NewGameCtx(
				m.game.resources, m.game.ecs, m.game.step,
				func(v any) { m.track(v) },
				func(producer func() []goke.System) { m.game.pendingSetup = append(m.game.pendingSetup, producer) },
			)
			err := p.Install(ctx)
			var nr *plugins.NotReadyError
			switch {
			case err == nil:
				progressed = true
				m.track(p)
			case errors.As(err, &nr):
				if ctx.Wrote() {
					panic(fmt.Sprintf("gokebiten: plugin %q wrote to Resources/ECS before returning an unmet dependency — Install must check its dependencies before any side effects", p.Name()))
				}
				if m.waitingOn == nil {
					m.waitingOn = make(map[string]reflect.Type)
				}
				m.waitingOn[p.Name()] = nr.Type
				stillPending = append(stillPending, p)
			default:
				return fmt.Errorf("gokebiten: install plugin %q: %w", p.Name(), err)
			}
		}
		m.pending = stillPending
		if len(m.pending) == 0 || !progressed {
			return nil
		}
	}
}

// finalizePending retries once more, then fails loudly if any plugin never became ready.
func (m *pluginManager) finalizePending() error {
	if err := m.resolvePending(); err != nil {
		return err
	}
	if len(m.pending) == 0 {
		return nil
	}
	reasons := make([]string, len(m.pending))
	for i, p := range m.pending {
		reasons[i] = fmt.Sprintf("%q waits on %s", p.Name(), m.waitingOn[p.Name()])
	}
	return fmt.Errorf("gokebiten: plugins never became ready (missing dependency or cycle among them): %s", strings.Join(reasons, "; "))
}

package gokebiten

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
)

// pluginManager installs plugins and tracks every module/provider they register, for Persistence.Load's auto-scan.
type pluginManager struct {
	game       *Game
	plugins    map[string]Plugin
	registered []any
}

// install runs p.Install once, rejecting a duplicate Name.
func (m *pluginManager) install(p Plugin) error {
	if m.plugins == nil {
		m.plugins = make(map[string]Plugin)
	}
	if _, dup := m.plugins[p.Name()]; dup {
		return fmt.Errorf("gokebiten: plugin %q already installed", p.Name())
	}
	ctx := &PluginContext{Resources: m.game.resources, game: m.game}
	if err := p.Install(ctx); err != nil {
		return fmt.Errorf("gokebiten: install plugin %q: %w", p.Name(), err)
	}
	m.plugins[p.Name()] = p
	return nil
}

// track records v so providedComps/postLoadSystems can find it later.
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

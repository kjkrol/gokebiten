package gokebiten

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/internal/persist"
)

// Persistence provides Save/Load/List for one Game's ECS and resources — see Game.Persistence.
type Persistence struct {
	game *Game
}

// PostLoader is implemented by a tracked module/provider needing a one-time system run right after Load restores entities.
type PostLoader interface {
	PostLoad() goke.System
}

// Saveable is implemented by a tracked plugin/module contributing its own
// state to every Persistence.Save/Load call, alongside whatever the caller
// passes explicitly.
type Saveable interface {
	SaveTargets() []any
}

// List returns every save found for basePath, "" (quicksave) first.
func (p *Persistence) List(basePath string) ([]string, error) { return persist.ListSaves(basePath) }

// Save writes resources and the ECS snapshot to disk under basePath/label, auto-including every tracked Saveable's targets.
func (p *Persistence) Save(basePath, label string, resources ...any) error {
	all := append(p.game.pluginManager.saveTargets(), resources...)
	return persist.Save(p.game.ecs, basePath, label, all...)
}

// Load restores a snapshot written by Save, auto-scanning tracked plugins for components, Saveable targets, and post-load systems.
func (p *Persistence) Load(basePath, label string, resources ...any) error {
	pm := p.game.pluginManager
	comps := pm.providedComps()
	all := append(pm.saveTargets(), resources...)
	if err := persist.Load(p.game.ecs, basePath, label, comps, all...); err != nil {
		return err
	}
	systems := pm.postLoadSystems()
	p.game.pendingSetup = append(p.game.pendingSetup, func() []goke.System { return systems })
	return nil
}

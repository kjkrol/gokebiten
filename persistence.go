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

// List returns every save found for basePath, "" (quicksave) first.
func (p *Persistence) List(basePath string) ([]string, error) { return persist.ListSaves(basePath) }

// Save writes resources and the ECS snapshot to disk under basePath/label.
func (p *Persistence) Save(basePath, label string, resources ...any) error {
	return persist.Save(p.game.ecs, basePath, label, resources...)
}

// Load restores a snapshot written by Save, auto-scanning tracked plugins for components and post-load systems.
func (p *Persistence) Load(basePath, label string, resources ...any) error {
	pm := p.game.pluginManager
	comps := pm.providedComps()
	if err := persist.Load(p.game.ecs, basePath, label, comps, resources...); err != nil {
		return err
	}
	systems := pm.postLoadSystems()
	p.game.pendingSetup = append(p.game.pendingSetup, func() []goke.System { return systems })
	return nil
}

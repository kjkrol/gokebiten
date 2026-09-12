package engine

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
)

// persistence is the only concrete implementation of game.Persistence.
type persistence struct {
	engine *Engine
}

var _ game.Persistence = (*persistence)(nil)

// List returns every save found for basePath, "" (quicksave) first.
func (p *persistence) List(basePath string) ([]string, error) { return listSaves(basePath) }

// Save writes resources and the ECS snapshot to disk under basePath/label, auto-including every tracked Serializable's targets.
func (p *persistence) Save(basePath, label string, resources ...any) error {
	return save(p.engine.ecs, basePath, label, p.engine.persistGroups(resources...))
}

// Load restores a snapshot written by Save, auto-scanning tracked plugins
// for components, Serializable targets, and post-load systems. A resource
// not present in the save (e.g. a plugin added since it was written) is
// left at its current value instead of failing the whole load.
func (p *persistence) Load(basePath, label string, resources ...any) error {
	comps := p.engine.providedComps()
	if err := load(p.engine.ecs, basePath, label, comps, p.engine.persistGroups(resources...)); err != nil {
		return err
	}
	p.engine.runRestore()
	systems := p.engine.postLoadSystems()
	p.engine.addPendingSetup(func() []goke.System { return systems })
	return nil
}

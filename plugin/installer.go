package plugin

import "github.com/kjkrol/goke/v3"

// Installer is what a Plugin gets during Install — ECS wiring only.
// Cross-plugin data/behavior comes from constructor injection instead.
type Installer interface {
	UseModule(m goke.Module)
	Setup(providers ...goke.SetupProvider)
	RegSys(factory func() goke.System) goke.Runnable
	ECS() *goke.ECS
}

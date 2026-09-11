# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**gokebiten** is a public, modular Go game engine library: a
user-implemented `game.Game` (`Init`/`Restore`/`Spawn`/`Update`/`Draw`/`HandleEvents`) is
driven by a `gokebiten.Engine` that wraps
[goke](https://github.com/kjkrol/goke) (a type-safe, archetype-based ECS)
into [Ebitengine](https://ebitengine.org/)'s `Update`/`Draw`/`Layout` loop.
Everything beyond the tick loop is installed as a `plugin.Plugin`, added
from `Game.Init` via `ctx.Use`.

Since this is a library third parties `go get` and browse on pkg.go.dev,
the extension contract — `game.Game`/`Initializer`/`Runtime`/`Persistence`
and `plugin.Plugin`/`Installer`/`Serializable`/`PostLoader` — lives in the
public root packages `game` and `plugin`, not `internal/`, so it gets full
godoc treatment. Only pure orchestration (`Engine` itself, nobody's godoc
a user needs to read) lives in `internal/engine`; root `gokebiten` just
re-exports `Engine`/`Props`/`NewEngine` as thin aliases over it.

## Commands

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./...   # standard verification sequence
go test ./plugins/world/... -run TestName -v                     # a single test
make demo                                                          # go mod tidy && run examples/collision-demo
make demo-nav                                                     # go mod tidy && run examples/board-navigation-demo
```

The two `examples/*` programs are real Ebitengine GUI apps (open a window)
— `go test` alone can't exercise them. To sanity-check one still runs after
a change in a headless environment: build to a temp path, run under
`timeout <n>s`, treat exit 124 (still running, not crashed) as healthy.
This doesn't substitute for looking at the demo when a visual change needs
actual confirmation.

CI (`.github/workflows/go.yml`) runs
`xvfb-run -a go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...`
on Go 1.27.0.

## Architecture

### Plugin system

`plugin.Plugin` — `Name`, `Install(ctx plugin.Installer) error`, `RunPlan`,
`WithRenderer`, `Renderer`, `EventHandler`, `Serializable` — is the one
extension point. A user's `Game` builds its plugins as its own struct
fields inside `Init` and installs each via `ctx.Use(p)`, which registers
`p.Serializable()` (if any) and calls `p.Install`. There is no dependency
retry mechanism: a plugin needing another plugin's *behavior* takes it as
an explicit constructor argument (e.g.
`collisions.NewPlugin(hitExpires, worldPlugin)`) rather than looking it
up — the dependency's construction order in the caller's code, not `Use`
registration order, is what matters. `Install` itself only queues ECS
wiring (`ctx.UseModule`/`ctx.Setup`), flushed once via a single
`ecs.Setup()` call after `Game.Init` returns — this is what lets
`Persistence.Load` decide fresh-spawn vs. restore before the ECS commits
to either path.

Package layout: `plugin` (root) — `Plugin`/`Installer`/`Serializable`/
`PostLoader`, the extension contract, zero internal dependencies. `game`
(root) — `Game`/`Initializer`/`Runtime`/`Persistence`, what a `Game`
implements and receives; imports `plugin` (`Initializer.Use(p
plugin.Plugin)`). `internal/engine` — the concrete `Engine` driver plus
the unexported `initializer`/`persistence`/`storage` implementing
`game.Initializer`/`game.Persistence`/the save registry; imports both
`game` and `plugin`. Dependency direction is one-way:
`plugin` ← `game` ← `internal/engine` ← `gokebiten`. Built-in plugins
(`plugins/*`) import `plugin` directly (not `gokebiten`), exactly like a
third-party plugin would.

### Module naming convention

Every package under `plugins/` that implements `goke.Module` names that
type `module` (unexported); the exported `Plugin` is a thin facade proxying
only what callers need. A `module` never also implements `goke.System`
directly — per-tick logic lives in its own dedicated type/file (e.g.
`selection.SelectionSystem`), which `module.RegSystems` constructs.

Each `plugin.go`/`module.go` groups methods under banner comments — contract
methods first, then everything plugin/module-specific — so a file's shape
shows how much of it is boilerplate vs. real behavior.

### Built-in plugins (`plugins/`)

- **`world`** — mandatory foundation: Position/Velocity/Appearance, entity
  spawning, the shared `*gokg.Space` index, per-tick movement.
- **`board`** — optional grid + terrain over `world`; depends on `world`.
- **`camera`** — publishes the shared `render.Camera`.
- **`collisions`** — optional broad/narrow-phase physics over `world`'s
  space; handler strategies live under `collisions/strategies/*`. Depends
  on `world`.
- **`navigation`** — pathfinding/movement toward a `MoveOrder` across a
  `board`. Depends on `board` and `world`.
- **`selection`** — mouse click/drag → `Selected` tag on `world` entities.
  Depends on `world`.

Each package has a `doc.go` describing the gameplay capability it adds.

### Game / persistence

`internal/engine.Engine` (aliased `gokebiten.Engine`) owns the `goke.ECS`
and the Ebitengine loop, and drives a user's `game.Game`. Its
`Persistence()` returns a `game.Persistence` (interface — implemented by
the unexported `internal/engine.persistence`, see `persistence.go` +
`persist.go`) implementing `Save`/`Load`/`List`; a tracked plugin/module
implementing `plugin.Serializable`/`plugin.PostLoader` is included
automatically. Saved resources are matched by name (a `Plugin`'s `Name()`,
or the Go type name for anything tracked without one) rather than by
position, so a save survives plugins being added/removed/reordered
between game versions — see `internal/engine/persist.go`.

### Testing conventions

Internal tests (`package world`, not `world_test`) are used when a test
needs unexported module state; otherwise use the external `_test` suffix.
Prefer testing through a plugin's real `Install` path over hand-built
shortcuts when what's under test is install-order or wiring behavior —
real regressions here have only shown up through the actual
`Install` → `ecs.Setup` → `RunPlan` sequence, not lower-level unit tests
that bypass it.

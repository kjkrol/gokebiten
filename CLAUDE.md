# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**gokebiten** is a modular Go game engine: a small `Game` core wraps
[goke](https://github.com/kjkrol/goke) (a type-safe, archetype-based ECS) into
[Ebitengine](https://ebitengine.org/)'s `Update`/`Draw`/`Layout` loop. Everything
beyond the tick loop is installed as a `plugins.Plugin` (`plugins/plugin.go`).

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

`plugins.Plugin` — `Name`, `Install(ctx *GameCtx) error`, `RunPlan`,
`WithRenderer`, `Renderer`, `EventHandler` — is the one extension point.
`Game.UsePlugin`/`Game.Init` wire plugins in; `pluginManager`
(`plugin_manager.go`) resolves `Install` calls, retrying any that return
`plugins.NotReadyError` until every plugin succeeds or the set is stuck.

A plugin needing another plugin's *behavior* takes it as an explicit
constructor argument (e.g. `collisions.NewPlugin(hitExpires, worldPlugin)`)
rather than looking it up — the dependency's construction order in the
caller's code, not `UsePlugin` registration order, is what matters.

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

### Game / persistence (root package `gokebiten`)

`game.go`'s `Game` owns the `goke.ECS` and the Ebitengine loop.
`persistence.go` + `internal/persist` implement `Game.Persistence.Save/Load`;
a tracked plugin/module implementing `gokebiten.Saveable`/`PostLoader` is
included automatically.

### Testing conventions

Internal tests (`package world`, not `world_test`) are used when a test
needs unexported module state; otherwise use the external `_test` suffix.
Prefer testing through a plugin's real `Install` path over hand-built
shortcuts when what's under test is install-order or wiring behavior —
real regressions here have only shown up through the actual
`Install` → `ecs.Setup` → `RunPlan` sequence, not lower-level unit tests
that bypass it.

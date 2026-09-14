# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**gokebiten** is a public, modular Go game engine library: a
user-implemented `game.Game` — a named collection of `game.Stage`s, each
with its own lifecycle (`Init`/`Restore`/`Spawn`/`Update`) and its own
`game.Scene`s (`Stack`/`Composition`, the sole entry point for input) — is
driven by a `gokebiten.Engine` that wraps
[goke](https://github.com/kjkrol/goke) (a type-safe, archetype-based ECS)
into [Ebitengine](https://ebitengine.org/)'s `Update`/`Draw`/`Layout` loop.
Everything beyond the tick loop is installed as a `plugin.Plugin`, added
from `Stage.Init` via `ctx.Use`. See "Stage / Scene" below for the model.

Since this is a library third parties `go get` and browse on pkg.go.dev,
the extension contract — `game.Game`/`Stage`/`Scene`/`Stack`/`Composition`/
`Initializer`/`Runtime`/`Persistence` and `plugin.Plugin`/`Installer`/
`Serializable`/`PostLoader` — lives in the public root packages `game` and
`plugin`, not `internal/`, so it gets full godoc treatment. Only pure
orchestration (`Engine` itself, nobody's godoc a user needs to read) lives
in `internal/engine`; root `gokebiten` just re-exports `Engine`/`Props`/
`NewEngine` as thin aliases over it.

## Commands

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./...   # standard verification sequence
go test ./plugins/world/... -run TestName -v                     # a single test
make demo                                                          # go mod tidy && run examples/collision-demo
make demo-nav                                                     # go mod tidy && run examples/board-navigation-demo
make demo-scenes                                                  # go mod tidy && run examples/scenes-demo
```

The three `examples/*` programs are real Ebitengine GUI apps (open a window)
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
extension point. A `Stage` builds its plugins as its own struct fields
inside `Init` and installs each via `ctx.Use(p)`, which registers
`p.Serializable()` (if any) and calls `p.Install`. There is no dependency
retry mechanism: a plugin needing another plugin's *behavior* takes it as
an explicit constructor argument (e.g.
`collisions.NewPlugin(hitExpires, worldPlugin)`) rather than looking it
up — the dependency's construction order in the caller's code, not `Use`
registration order, is what matters. `Install` itself only queues ECS
wiring (`ctx.UseModule`/`ctx.Setup`), flushed once via a single
`ecs.Setup()` call after `Stage.Init` returns — this is what lets
`Persistence.Load` decide fresh-spawn vs. restore before the ECS commits
to either path. The same generic `ctx.Track(s plugin.Serializable)` lets a
`Stage` register non-`Plugin` state (its `Composition`, see "Stage /
Scene") into the same save/load machinery, without it needing to be a full
`Plugin`.

Initial state follows the same optional-interface pattern as
`plugin.Restorer`/`plugin.PostLoader`: `Stage.Spawn` only declares it through
each plugin's own typed `Seed` (`world.Plugin.Seed(roster)`,
`board.Plugin.Seed(layout)`), and the engine then calls `Populate()` on every
tracked `plugin.Populator` — only when `Restore` loaded nothing. Entity kinds
(`world.EntKind`: every component as `Const` or `Load` from a roster entry's
`Data`) and cell kinds are registered in `Stage.Init` via
`EntKindDict()`/`CellKindDict()`, which also issue their `SpriteID`s.

Package layout: `render` (root) — `Renderer`/`AtlasSource`/`Atlas`/
`CachedRenderer`/`QuadBatch`/`SolidBackground`/`TelemetryRenderer`, pure
drawing primitives, zero knowledge of Stages/Scenes/plugins. `plugin`
(root) — `Plugin`/`Installer`/`Serializable`/`PostLoader`/`Populator`, the extension
contract; imports `render` (`Plugin.WithRenderer(atlas
render.AtlasSource)`). `game` (root) — `Game`/`Stage`/`Scene`/`Stack`/
`Composition`/`Initializer`/`Runtime`/`Persistence`, what a `Stage`
implements and receives, all in one package (Scene needs the same
`Runtime` a Stage does, so they're never split across a layering boundary
— see "Stage / Scene"); imports `plugin` and `render`. `internal/engine`
— the concrete `Engine` driver plus the unexported `ecsHost`/
`initializer`/`persistence`/`storage`/`stageRuntime` implementing
`game.Initializer`/`game.Persistence`/the save registry/one active
`Stage`'s runtime; imports `game`, `plugin`, and `render`. Dependency
direction is one-way: `render` ← `plugin` ← `game` ← `internal/engine` ←
`gokebiten`. Built-in plugins (`plugins/*`) import `plugin`/`render`
directly (not `gokebiten`), exactly like a third-party plugin would.

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

- **`world`** — foundation a Stage installs by calling
  `ctx.UseWorld(cfg)` in `Init`, once (a second call panics); `cfg` sizes
  the space, toroidality, entity bounds and camera, and a Stage that never
  calls it gets no world:
  Position/Velocity/Appearance, entity spawning, the shared `*gokg.Space`
  index, per-tick movement, and the shared `camera.Camera` (a root package,
  not a plugin of its own) exposed via `world.Plugin.Camera()`.
- **`board`** — optional grid + terrain over `world`; depends on `world`.
- **`collisions`** — optional broad/narrow-phase physics over `world`'s
  space; handler strategies live under `collisions/strategies/*`. Depends
  on `world`.
- **`navigation`** — pathfinding/movement toward a `MoveOrder` across a
  `board`. Depends on `board` and `world`.
- **`selection`** — mouse click/drag → `Selected` tag on `world` entities.
  Depends on `world`.

Each package has a `doc.go` describing the gameplay capability it adds.

### Stage / Scene

A `game.Game` also supplies `Props()` (window/tick-rate config, read
once at startup by `gokebiten.Run(g)`) alongside a named collection of
`game.Stage`s plus which one starts active (`Stages() (map[string]Stage,
string)`) — every game writes its own small `Game` implementation, even
for a single Stage, since only a concrete type can supply its own `Props`.
A `Stage` is what `Game` itself used to be:
`Init`/`Restore`/`Spawn`/`Update`, each with its **own** `*goke.ECS`, built
fresh (plus a fresh `world.Plugin` if its `Init` calls `ctx.UseWorld`; the
engine only fills an unset camera viewport from the screen size) the moment `Runtime.SwitchStage`
enters it — so a menu Stage can sit idle with zero gameplay entities until
the player actually starts the game, at which point the gameplay Stage's
`Init`/`Restore`/`Spawn`/`ecs.Setup` run for the first time. A `Stage` is
never a `plugin.Plugin` — it's the thing that *installs* plugins via
`ctx.Use`, exactly like `Game` did before. **A `Stage` has no
`HandleEvents`** — input is a `Scene`'s sole responsibility (see below); a
shortcut a game wants shared across multiple Scenes (e.g. Escape-to-quit)
is a plain helper function each of their `HandleEvents` calls, not an
engine concept.

Within one active `Stage`, `game.Scene` is what `Game.Draw`/`HandleEvents`
used to be: `Name`, `Layers() []func() render.Renderer`, `HandleEvents`,
`Focusable`. `Stage.Stack()` is the static, `Name()`-keyed registry of
every `Scene` it can show (`game.NewStack(scenes...)`); `Stack.Composition()`
(the only way to reach it — `Stage` has no accessor of its own) is the live
per-tick state over that Stack — which scenes are visible, in
what z-order, and which one is `Active()` (the topmost with `Focusable()
== true`, so a non-focusable HUD/minimap can sit on top and still never
steal input). Every tick, the engine calls `HandleEvents` on **only** the
Scene `Composition.Active()` names — never in parallel with anything at
the `Stage` level. `Composition` embeds `plugin.Serializable` directly (no
import trickery needed — `game` already imports `plugin`) — a `Stage`
registers it once via `ctx.Track(composition)` in `Init` so visibility/
order survive `Persistence.Save`/`Load` automatically, the same name-keyed
way a `Plugin`'s own state does.

`Runtime` (`Paused`/`Pause`/`Resume`/`TogglePause`/`Quit`/`SwitchStage`/
`Persistence`/`TPS`/`Camera`) is a single, undivided interface — the exact
same value reaches both a `Stage` (via `Initializer`/at Restore time) and
every `Scene`'s `HandleEvents`; there is no cut-down "scene-level" subset.
A menu scene's "Start" button calls `runtime.SwitchStage(gameplayStage.
Name())` directly, no need to bubble a click up through anything. Anything
that's a *plugin's* capability rather than an engine primitive (camera,
selection, navigation...) reaches a `Scene` the same way it reaches a
sibling `Plugin`: constructor injection at `Stage.Init` time, as a plain
struct field — never through `Runtime`.

See `examples/scenes-demo` for a full walkthrough: a menu `Stage` with no
gameplay entities, "Start" switching (lazily building the ECS) into a
gameplay `Stage`, and a togglable modal `Scene` over the still-ticking
world plus a non-focusable HUD overlay.

### Game / persistence

`internal/engine.Engine` (aliased `gokebiten.Engine`) owns the Ebitengine
loop and drives a user's `game.Game` one active `Stage` at a time — see
"Stage / Scene" above. `Engine` never caches its own copy of the Stage
set: it calls `game.Stages()` whenever it needs to resolve a name (once in
`Init`, and on every `Runtime.SwitchStage`) — `game.Game` is the sole
container. Its `Persistence()` returns a `game.Persistence` (interface —
implemented by the unexported `internal/engine.persistence`, see
`persistence.go` + `persist.go`) bound to the active `Stage`'s own
`*goke.ECS`, implementing `Save`/`Load`/`List`; a tracked plugin/module
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

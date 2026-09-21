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
make demo-vision                                                  # go mod tidy && run examples/vision-demo
```

The four `examples/*` programs are real Ebitengine GUI apps (open a window)
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
`WithRenderer`, `Renderer`, `EventHandler`, `Serializable`,
`RegisterBehavior` — is the one extension point. A behavior is registered on
the plugin it concerns and run inside that plugin's own pass. `plugin.Between[A,
B]` (a pair of tags) and `plugin.Each[T]` (one entity) build them; the tags join
the host's queries as optional components, so a behavior costs no query. The
func's payload type — `collision.Meeting`, `collision.Struck`,
`vision.Sighting` — is what says whose it is: a host refuses one made for another (`ErrUnhostedBehavior`), so
registering in the wrong place is an error, never a silent no-op. A plugin
hosts them with `plugin.PairHost[P]`/`plugin.EachHost[P]`. A `Stage` builds its plugins as its own struct fields
inside `Init` and installs each via `ctx.Use(p)`, which registers
`p.Serializable()` (if any) and calls `p.Install`. There is no dependency
retry mechanism: a plugin needing another plugin's *behavior* takes it as
an explicit constructor argument (e.g.
`collision.NewPlugin(hitExpires, worldPlugin)`) rather than looking it
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
are defined in `Stage.Init` with the `plugins/world/kind` package:
`prey := kind.Define[P](world.Kinds(), "prey", kind.Spec{...})` — a `Spec` is
just the list of a kind's components, each `kind.Const(v)` (same for all) or
`kind.Load(func(row P) T)` (read from that entity's row), `world.Position` and
`world.Velocity` among them (one of each, or `Define` panics by name; so does a
`Load` over a row type other than `P`). `Define` hands back the kind itself,
`kind.Of[P]`: `prey.Entry(row)` builds a roster entry for `world.Plugin.Seed`
— the row's type is checked by the compiler — and `prey.ID()`/`SpriteID()` say
what its entities carry and are drawn from. `kind` never imports `world`
(`world` imports it), which is why a kind's id is `kind.ID` and the registry,
`world.Kinds`, sits behind the `kind.Registry` interface; it also issues atlas
slots no kind owns (`NewSprite`) and tells `Persistence.Load` about
every component type its kinds carry (`Kinds.LoadComps`), so a game's own tags
and state (`behavior.Predator`, `behavior.HitMark`) survive a save without being registered
anywhere else; the engine lists a type a kind shares with a module once. Cell
kinds go through `board.Plugin.CellKindDict().Create`.

A `render.Atlas` sizes nothing up front: `Register(size, draw)`/`RegisterAt(id,
size, draw)` only record sprites, each at a texture size of its own (the drawn
size is the entity's box; this is resolution), and `Close()` is what lays the
sheet out and bakes it — so a slot issued late (`Kinds().NewSprite()`) is as
welcome as an early one, as long as it comes before `Close`.

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
  calls it gets no world. `SpaceCfg.Edges` (`aabbworld.Edges`) sets the edge rule
  per axis — `aabbworld.Torus`, `WrapX`/`WrapY` alone, `OpenX`/`OpenY`, a closed
  axis by default: a box stops whole at a closed edge, wraps at a wrapping one,
  and may leave by an open one. An entity wholly past an open edge is dropped
  from the index and handed, once, to `world.Plugin.OnExit(fn)` — despawned when
  no handler is set; a sibling plugin that moves boxes itself (collision's solver)
  reports through `world.Plugin.Tracked`:
  `Base` — the one component every entity carries, holding its `Position`,
  `Velocity` and `TypeID`, so a host hands it to whatever it hosts instead of
  anyone binding it twice — plus Appearance, entity spawning and `Despawn` (which clears
  both the ECS and the index, so nothing goes on seeing a ghost),
  `Attach(cb, id, v)`/`Detach[T](cb, id)` — the mid-game counterparts of a
  kind's `k.Const`, for game logic that has a `plugin.Tick` and no `CompID`
  (`Declare[T]()` in `Stage.Init` tells saves about a type only ever attached) — the shared
  `*aabbworld.Space` index, per-tick movement — capped per entity at half its own
  shorter side (`world.StepReach`, `Position.MaxStep`/`MaxSpeed`), so mixed
  sizes share a world without the smallest slowing the rest — and the shared `camera.Camera` (a
  root package, not a plugin of its own; it keeps its own window arithmetic —
  wrapping on a wrapping axis, held inside the world on any other) exposed via
  `world.Plugin.Camera()`.
- **`board`** — optional grid + terrain over `world`; its grids wrap per axis,
  following the world's `Edges` (`SetWrap(x, y)`). Depends on `world`.
- **`collision`** — optional broad/narrow-phase detection over `world`'s
  space. An entity collides exactly while it carries `Collider` —
  `kind.Const(collision.Collider{})`, or `Attach`/`Detach` mid-game; the plugin
  tells the index itself, either way. The broad phase is one `collide.BroadPhase(space, world.StepReach,
  aabbworld.CanCollide, …)` a tick — every pair whose reaches touch, once — into a
  `Candidates` list the narrow phase resolves by `Seek` and hands, as `collide.Pair`s, to a
  `collide.NarrowPhase` whose `Separate` tests each exactly, pushes the overlapping apart and tells
  the index where they came to rest — `Left()` names whoever it pushed out through an open edge
  (both from `github.com/kjkrol/aabbworld/collide`). An entity carrying `Physics` (`Mass`, `Restitution` 0–1) is pushed out
  of overlaps and bounces — the bounce is the engine's own, an infinite `Mass`
  is a wall; one without `Physics` is only ever detected (a town, a trigger).
  Separation is always an even split. Reactions are behaviors hosted inside the
  engine's own passes: `plugin.Between[A, B]` of a `Meeting` per contact between
  two tags (narrow phase, `plugin.Anything` as the wildcard), `plugin.Each[T]`
  of a `Struck` per entity per tick (broad phase). A strategy exports a plain
  function of the flat `collision/behavior` package (`CountContacts`, `LogContacts`, `ShowHits`) — the tags it runs between
  are named where it is registered, `RegisterBehavior(plugin.Between[A, B](fn), ...)`.
  `Collider` is the plugin's one aggregate: this tick's candidates plus what the
  entity struck (`Collider.Contacts()`). Depends on `world`.
- **`navigation`** — pathfinding/movement toward a `MoveOrder` across a
  `board`. Depends on `board` and `world`.
- **`selection`** — mouse click/drag → `Selected` tag on `world` entities.
  Depends on `world`.
- **`vision`** — narrowed perception: a `Sight` cone scanned against `world`'s
  space each tick fills its own `Sight.Seen` (who this entity can see, nearest first), and
  `SightOutline` on an entity gets its view's shape computed and drawn. It
  hosts `plugin.Between[A, B]` of a `Sighting` inside the scan's own pass: once
  a tick per observer carrying `A`, with everything in view carrying `B` — a
  directed pair, grouped by observer, empty included. A behavior tells its seen
  entities apart with `plugin.Asking[T]` + `Seen.Carries[T]()`, and steers only
  through `Steering.Request`. Ready-made ones live in the flat `vision/behavior`
  package as plain functions (`Flee.Steer`, `Chase`); a file using both plugins'
  behaviors imports them as `cbehavior`/`vbehavior` — who flees or hunts
  whom is the registration's to say. Depends on `world`.

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
used to be: `Name`, `Layers() []render.Renderer` (built once, on entering the Stage; the engine only calls `Draw` per frame), `HandleEvents`,
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

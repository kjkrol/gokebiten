# gokebiten

<p align="center">
  <img src=".github/docs/img/logo.png" alt="GOKe Logo" width="300">
  <br>
  <a href="https://go.dev">
    <img src="https://img.shields.io/badge/Go-1.27.0+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  </a>
  <a href="https://pkg.go.dev/github.com/kjkrol/gokebiten">
    <img src="https://img.shields.io/badge/GoDoc-Reference-007d9c?style=flat-square&logo=go" alt="GoDoc">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="License">
  </a>
  <a href="https://app.codecov.io/gh/kjkrol/gokebiten">
    <img src="https://img.shields.io/codecov/c/github/kjkrol/gokebiten?style=flat-square&logo=codecov" alt="Codecov Coverage">
  </a>
  <a href="https://github.com/kjkrol/goke/actions">
    <img src="https://github.com/kjkrol/gokebiten/actions/workflows/go.yml/badge.svg" alt="Go Quality Check">
  </a>
</p>

**gokebiten** integrates [goke](https://github.com/kjkrol/goke) — a type-safe, archetype-based
Entity Component System for Go — with [Ebitengine](https://ebitengine.org/). It wraps goke's
`ECS`/`System`/`Plan` model behind a small `Game` type that implements `ebiten.Game`, and layers
spatial indexing (via [GOKg](https://github.com/kjkrol/gokg)), kinematics, and collision
resolution on top.

## What's here

* `Game`/`Resources`/`Plugin` (repo root) — wires a goke `ECS` into Ebitengine's `Update`/`Draw`/
  `Layout` loop, behind a deliberately small public API (a dozen or so methods covering lifecycle,
  save/load, and plugging things in — nothing more). `Resources` is a typed registry (methods
  `InsertResource[T]`/`GetResource[T]`/`TryGetResource[T]`, keyed by type — in the spirit of Bevy
  ECS's `Res<T>`) for anything that isn't ECS state: `*GameProps`, input, telemetry, game-owned
  state, and whatever else a `Plugin` contributes.
* `Plugin` — the unit of extension, and the *only* way to add functionality: one
  `Install(ctx *GameCtx)` call wires an ECS module, one-time setup, renderers, and/or
  resources together as a single, swappable piece, installed via `Game.UsePlugin`. The
  ECS-registration primitives (`UseModule`/`Setup`/`RegSys`/raw `*goke.ECS` access) are reachable
  only through `GameCtx`, not on `Game` itself — bypassing a `Plugin` isn't possible.
  `world.Plugin`, `physics.Plugin`, and `camera.Plugin` are the built-in ones (see below);
  writing your own plugin package needs nothing beyond this repo's public API. `Plugin` itself is a
  minimal contract (`Name`/`Install`) — any plugin-specific configuration (e.g.
  `physics.Plugin`'s `SetCollisionHandlers`) lives on that plugin's own concrete type, via fluent
  `NewPlugin(...).With...`/`Set...` methods, never exported struct fields.
* `control/` — input capture and event dispatch.
* `physics/` — kinematics (movement, boundary handling) and collision detection/resolution
  (broad phase via `gokg.Space`, narrow phase, pluggable `CollisionHandler` strategies), exposed as
  `physics.Plugin`.
* `render/` — sprite batching, a small `Camera` interface (`ToScreen`/`Visible`/`Bounds`/
  `ExtendedBounds`) plus its default `BasicCamera` implementation (wrap/clamp-aware, with zoom),
  tag-driven overlays, telemetry HUD.
* `world/` — population/spawn bookkeeping on top of a `gokg.Space`, exposed as `world.Plugin`.
* `camera/` — `camera.Plugin`, the default way to get a shared `render.Camera`: sized from
  `GameProps` (or an explicit viewport), wrap-aware when a `world.Plugin` is installed first.
  Nothing renderer-side depends on this package — any type implementing `render.Camera` works,
  built-in or your own.

**Note:** this is an evolving, pre-1.0 API — `Resources[S, T]` (a fixed two-slot generic bundle)
was replaced by the typed `Resources` registry above, `render.Camera` was later split into an
interface (`Camera`) plus a default implementation (`BasicCamera`), and `Resources`' accessors
moved from free functions to generic methods. Game state persisted via `Game.Save`/`Game.Load`'s
variadic `resources` parameters is gob-encoded, so save files from before these changes are not
compatible with the current version.

## Installation

```bash
go get github.com/kjkrol/gokebiten
```

## Example

[**examples/collision-demo**](./examples/collision-demo/main.go) — a real-time simulation of
thousands of moving, colliding AABBs at a fixed 120 TPS, built entirely on this package plus
goke's archetype-based storage and parallel systems.

<table>
  <thead>
    <tr>
      <th style="text-align: left; vertical-align: top; width: 400px;">
        <video src="https://github.com/user-attachments/assets/2b921500-eb3e-49bf-98ee-ac741746e64d" width="400" autoplay loop muted playsinline></video>
        <br>
          <sub><strong>Stats:</strong> 2306 colliding AABBs | 120 TPS | 50 collisions/tick</sub>
      </th>
      <th style="text-align: left; vertical-align: top; width: 400px;">
        <video src="https://github.com/user-attachments/assets/50695c5a-4f77-4352-87da-1fa13168415b" width="400" autoplay loop muted playsinline></video>
        <br>
        <sub><strong>Stats:</strong> 524 colliding AABBs | 120 TPS | 15 collisions/tick</sub>
      </th>
    </tr>
  </thead>
</table>

Run it locally:

```bash
make run
```

## Prerequisites

* Go 1.27+
* [Ebitengine dependencies](https://ebitengine.org/en/documents/install.html) (C compiler and
  system libraries — Ebitengine uses cgo on most platforms)

## Relationship to goke

This package used to live inside `goke`'s `examples/ebiten-demo`; it has been extracted into its
own repository so `goke`'s core stays free of GUI dependencies while this integration can evolve
(and version) independently. See [goke](https://github.com/kjkrol/goke) for the ECS engine itself.

## License

MIT — see [LICENSE](./LICENSE).

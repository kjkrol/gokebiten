# Views: from a rectangle of the world to a player, to a network

[← Back to README](../README.md)

> A design sketch, not a contract: what the layers are, who owns what, and what each one leaves
> to the next. Code exists for the first layer only.

## 1. `world.View` — a rectangle and the entities in it

A `View` is what one pair of eyes sees: a rectangle of the world (`Bounds`) and the entities the
Space finds in it (`In`, an `EntitySet` — a set of the world's entities by index). The world keeps
any number of Views; each is made over a source of bounds (`world.Plugin.NewView(func() geom.AABB)`)
and refreshed by the `ViewSystem` once a tick, right after movement has rebuilt the Space. A View
whose bounds cover the whole world is not queried and simply sees everything; so does the zero
View a Stage has before its first tick.

The camera's View (`Plugin.View()`) is made by the world plugin itself, and the entity renderer
draws what it contains and nothing else. A View knows neither who is looking nor why.

Why a set and not a list: a set of entity indices masks a sequential walk over the ECS — one bit
test per entity, the real work only for the entities in view — and two Views combine as bit
operations (who is in both, who is in one and not the other) without any deduplication.

## 2. Player views — a plugin for whoever looks at the screen

A *player* is a pair of eyes with a place on the screen: its own `camera.Camera` (its own
`State`, saved with the game), a `world.View` over that camera's bounds, the rectangle of the
screen it is shown on, and a source of input (a keyboard or a pad here; the network below).
Two players on one screen is split screen: two cameras, two rectangles, one world, one ECS.

What this needs from the layers below: the entity renderer parameterised by a View and a camera
and aimed at a screen rectangle (an offscreen image per player, or an offset in `ToScreen` — to be
decided when this layer is planned). Today's `world.Renderer` with one camera and the whole screen
is the one-player case of this. Selection, camera controls and every other input handler become
per player, since each acts through a player's camera.

A player is a plugin (`plugins/players`), because a game that has exactly one player looking at
the whole screen should not have to know about any of this.

## 3. Networking — a plugin over player views

The server is one instance of the engine: Stages, ECS, plugins, ticking as always. A remote
client is a player whose view is shown elsewhere: its bounds arrive over the wire (the client
sends its `camera.State`) and its View, instead of going to a renderer, goes into a frame.

- **Deltas from the View.** The View keeps last tick's set beside this tick's: entities in the new
  set and not the old *entered* the view (the client creates a sprite), the reverse *left* (the
  client drops it), the rest are *updated* (positions). A client that just joined gets everything
  as entered.
- **One walk for all clients.** One pass over `Base`+`Appearance` serves every client: for up to 64
  clients a `uint64` mask per entity (bit = that client's View contains it), and the entity is
  appended to the frame of each client in its mask. More clients group into more masks. It is the
  renderer's principle again: a set masks a sequential walk; no list per client.
- **A frame** per tick per client: the tick number, the client's camera state echoed back,
  `entered: [id, kind, sprite, box]`, `updated: [id, box]`, `left: [id]`. Binary, little-endian;
  quantised positions later, if bandwidth asks.
- **The client** has no ECS and no Space: a camera, an atlas (kinds and their sprites are agreed
  when it joins), a `render.QuadBatch`, frames coming in, input going out (its camera state and
  the game's own commands). The transport sits behind an interface so tests run through memory.
- **Open.** Who has authority over input; the server's tick against the client's frame rate
  (interpolation); joining mid-game (a snapshot); trust (a LAN to begin with).

Networking is a plugin (`plugins/netview`), installed only by a game that wants it; the world and
the player layer know nothing about it.

## What each layer promises the next

| Layer | Owns | Promises upward |
|:---|:---|:---|
| `world.View` | the Space query, the set, when it is refreshed | "these entities are in these bounds, as of this tick" |
| players | cameras, screen rectangles, input per player, saving cameras | "this player sees this View, through this camera" |
| netview | frames, deltas, transport, the client | "this player is somewhere else" |

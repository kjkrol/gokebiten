// Package host is what a plugin that runs behaviors needs, and a game never imports: the
// constructors behind a plugin's own Between, Each and Every, and the hosts that run them.
//
// # Constructors
//
// [Pair] reacts to every pair a host meets where one entity carries tag a and the other b
// (plugin.Any for either); [Each] reacts on every entity the host visits that carries T; [Every]
// on all of them. A plugin exposes them typed to its payload — vision.Between(a, b, fn) for a
// Sighting, board.Each[T](fn) for a Standing — so a game names the plugin, never this package.
//
// # Hosts
//
// [PairHost] runs Pair behaviors: Bind its families to the host's queries once, then read what an
// entity carries (InChunk, At) as plugin.Marks and Dispatch, DispatchEitherWay or DispatchGrouped
// per pair or per observer. [EachHost] runs Each and Every behaviors: Bind, then Run over each
// chunk walked with a function describing its i-th entity. Both refuse another payload's behavior
// with plugin.ErrUnhostedBehavior and a late one with plugin.ErrHostBuilt.
package host

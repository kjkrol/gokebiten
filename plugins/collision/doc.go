// Package collision detects overlaps between world entities each tick and records what each
// struck on its Collider. An entity takes part while it carries Collider; one also carrying
// Physics is pushed apart and bounces. Reactions are behaviors: plugin.Between of a Meeting,
// plugin.Each of a Struck, registered with Plugin.RegisterBehavior.
package collision

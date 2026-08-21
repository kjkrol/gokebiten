package collisions

// Static marks an entity NarrowPhase must never move during MTV resolution
// — e.g. level geometry with no Velocity. resolveB checks this tag directly
// instead of probing which Query matches: Query.Seek is documented to
// bypass the archetype mask, so a probe like "try dynQry.Seek, fall back to
// staticQuery.Seek" always succeeds on the first try regardless of whether
// the entity actually has a Velocity component, silently reading garbage
// bytes as one.
type Static struct{}

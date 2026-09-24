// Package behavior holds ready-made reactions to contacts, each a plain function handed to
// plugin.Between or plugin.Each: CountContacts for telemetry, ShowHits with HitOverlay to
// keep a hit visible, LogContacts to write a line per contact.
//
// # CountContacts
//
// [CountContacts] adds every Meeting it is handed to a [ContactStats] the game owns. The total
// only grows; whoever shows a rate works it out from it, as render.TelemetryRenderer does.
//
// # ShowHits and HitOverlay
//
// [ShowHits] is a plugin.Each over [HitMark]: an entity that struck something shows a hit for its
// own Duration, or the one given. [HitOverlay] is a Drawing behavior for the world plugin: an
// overlay sprite on top of the entity while its HitMark is active.
//
// # LogContacts
//
// [LogContacts] writes a line per contact to stdout in the default format; [LogTo] and [LogAs]
// change where and how.
package behavior

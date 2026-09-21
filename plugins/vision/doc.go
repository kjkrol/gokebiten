// Package vision gives entities a narrowed view of the world: a Sight cone sees what falls
// inside it, within range and not hidden behind something nearer. Each tick fills Sight.Seen
// and runs plugin.Between behaviors of a Sighting; SightOutline gets the view drawn.
package vision

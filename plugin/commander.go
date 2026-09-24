package plugin

import "github.com/kjkrol/gram/control"

// Commander is a plugin, or a game, that defines commands: it lists the inboxes they land in and
// the bindings it suggests for them — possibly none. The players plugin is built over Commanders.
type Commander interface {
	Commands() []control.Mailbox
	DefaultBindings() []control.Binding
}

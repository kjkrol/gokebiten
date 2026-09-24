package control

import "reflect"

// PlayerID names a player of the game; Nobody is a command nobody in particular gave.
type PlayerID uint8

// Nobody is the PlayerID of a command with no player behind it — a test, a script.
const Nobody PlayerID = 0

// Issued is one command as its owner reads it: who gave it and what it says.
type Issued[C any] struct {
	Player  PlayerID
	Command C
}

// Inbox holds the commands of one type until the plugin that owns it drains them in its own pass;
// the owner keeps it as a field and lists it in its plugin.Commander's Commands.
type Inbox[C any] struct{ items []Issued[C] }

// Mailbox is an Inbox with its command type erased, as a carrier sorts commands into them.
type Mailbox interface {
	Accepts() reflect.Type
	Put(player PlayerID, cmd any)
	Clear()
}

var _ Mailbox = (*Inbox[struct{}])(nil)

func (in *Inbox[C]) Accepts() reflect.Type        { return reflect.TypeFor[C]() }
func (in *Inbox[C]) Put(player PlayerID, cmd any) { in.Add(player, cmd.(C)) }
func (in *Inbox[C]) Clear()                       { in.items = in.items[:0] }

// Add puts cmd in as given by player.
func (in *Inbox[C]) Add(player PlayerID, cmd C) { in.items = append(in.items, Issued[C]{player, cmd}) }

// Drain hands every command to fn in the order given and empties the inbox.
func (in *Inbox[C]) Drain(fn func(i Issued[C])) {
	for _, it := range in.items {
		fn(it)
	}
	in.items = in.items[:0]
}

// Empty reports whether nothing is waiting.
func (in *Inbox[C]) Empty() bool { return len(in.items) == 0 }

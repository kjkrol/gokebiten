package players

// Issued is one command as its owner reads it: who gave it and what it says.
type Issued[C any] struct {
	Player  *Player // nil for a command nobody in particular gave, as in tests
	Command C
}

// Inbox holds the commands of one type until the plugin that owns it drains them in its own pass.
type Inbox[C any] struct{ items []Issued[C] }

// mailbox is an Inbox with its command type erased, as the plugin sorts commands into them.
type mailbox interface {
	put(player *Player, cmd any)
	clear()
}

func (in *Inbox[C]) put(player *Player, cmd any) { in.Add(player, cmd.(C)) }
func (in *Inbox[C]) clear()                      { in.items = in.items[:0] }

// Add puts cmd in as given by player; Plugin.Issue is the usual way, this the direct one.
func (in *Inbox[C]) Add(player *Player, cmd C) { in.items = append(in.items, Issued[C]{player, cmd}) }

// Drain hands every command to fn in the order given and empties the inbox.
func (in *Inbox[C]) Drain(fn func(i Issued[C])) {
	for _, it := range in.items {
		fn(it)
	}
	in.items = in.items[:0]
}

// Empty reports whether nothing is waiting.
func (in *Inbox[C]) Empty() bool { return len(in.items) == 0 }

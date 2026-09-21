package collision

import "github.com/kjkrol/uid"

// Candidate is two entities close enough to be worth a real overlap test, the lower index first.
type Candidate struct{ A, B uid.UID64 }

// Candidates is what the broad phase hands the narrow phase each tick — build
// one and give it to both when running the two phases outside the plugin.
type Candidates struct{ pairs []Candidate }

// Add records one pair for the narrow phase to test.
func (c *Candidates) Add(a, b uid.UID64) { c.pairs = append(c.pairs, Candidate{A: a, B: b}) }

// All is every pair found since the broad phase last ran.
func (c *Candidates) All() []Candidate { return c.pairs }

// reset empties the list for a new tick, keeping the memory.
func (c *Candidates) reset() { c.pairs = c.pairs[:0] }

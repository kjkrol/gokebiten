// Package timing implements Game's fixed-timestep loop bookkeeping.
package timing

import "time"

// Tracker is responsible for the fixed physics step (Fixed Time Step) and statistics.
type Tracker struct {
	lastUpdate    time.Time
	lastTPSUpdate time.Time
	accumulator   time.Duration
}

func New() *Tracker {
	now := time.Now()
	return &Tracker{
		lastUpdate:    now,
		lastTPSUpdate: now,
	}
}

// CalculateSteps calculates how many physics ticks should be performed in the current frame.
func (t *Tracker) CalculateSteps(physicsStep time.Duration, maxSteps int) int {
	now := time.Now()

	if t.lastUpdate.IsZero() {
		t.lastUpdate = now
		t.lastTPSUpdate = now
	}

	elapsed := now.Sub(t.lastUpdate)
	t.lastUpdate = now
	t.accumulator += elapsed

	steps := 0
	for t.accumulator >= physicsStep && steps < maxSteps {
		t.accumulator -= physicsStep
		steps++
	}

	if t.accumulator > physicsStep {
		t.accumulator = 0
	}

	return steps
}

func (t *Tracker) ProcessStatsInterval() bool {
	duration := time.Since(t.lastTPSUpdate)
	if duration > 2*time.Second {
		t.lastTPSUpdate = time.Now()
		return true
	}
	if duration >= time.Second {
		t.lastTPSUpdate = t.lastTPSUpdate.Add(time.Second)
		return true
	}
	return false
}

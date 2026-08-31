package board

import (
	"slices"

	"github.com/kjkrol/uid"
)

// Occupancy tracks which entity/entities hold each cell, gating and
// recording every step Movement takes.
type Occupancy interface {
	CanEnter(c CellID, entity uid.UID64) bool
	Enter(c CellID, entity uid.UID64)
	Leave(c CellID, entity uid.UID64)
}

// SingleOccupancy allows at most one entity per cell, rejecting every
// other entrant. Zero-value ready — no constructor needed.
type SingleOccupancy struct {
	holders map[CellID]uid.UID64
}

var _ Occupancy = (*SingleOccupancy)(nil)

func (o *SingleOccupancy) CanEnter(c CellID, entity uid.UID64) bool {
	holder, occupied := o.holders[c]
	return !occupied || holder == entity
}

func (o *SingleOccupancy) Enter(c CellID, entity uid.UID64) {
	if o.holders == nil {
		o.holders = make(map[CellID]uid.UID64)
	}
	o.holders[c] = entity
}

func (o *SingleOccupancy) Leave(c CellID, entity uid.UID64) {
	if o.holders[c] == entity {
		delete(o.holders, c)
	}
}

// MultipleOccupancy lets any number of entities share a cell. Zero-value
// ready — no constructor needed.
type MultipleOccupancy struct {
	holders map[CellID][]uid.UID64
}

var _ Occupancy = (*MultipleOccupancy)(nil)

func (o *MultipleOccupancy) CanEnter(CellID, uid.UID64) bool { return true }

func (o *MultipleOccupancy) Enter(c CellID, entity uid.UID64) {
	if slices.Contains(o.holders[c], entity) {
		return
	}
	if o.holders == nil {
		o.holders = make(map[CellID][]uid.UID64)
	}
	o.holders[c] = append(o.holders[c], entity)
}

func (o *MultipleOccupancy) Leave(c CellID, entity uid.UID64) {
	list := o.holders[c]
	for i, e := range list {
		if e == entity {
			o.holders[c] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

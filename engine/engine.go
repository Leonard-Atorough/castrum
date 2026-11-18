// Package engine declares engine wiring as data: subsystem [Registration]
// values that the root castrum package applies in order when building a
// game. Built-in subsystems fill in the same structure, so engine wiring and
// future external wiring share one contract.
package engine

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/serialization"
)

// Slot is where in a phase's schedule a system runs. Slots execute in the
// order Opening, First, Regular, Last, Closing.
type Slot int

const (
	// Regular runs after previously registered regular systems in the
	// phase. It is the zero value, so a spec that names no slot runs here.
	Regular Slot = iota
	// Opening runs before everything else in the phase. At most one per phase.
	Opening
	// First runs after the opening entry and before regular systems.
	First
	// Last runs after regular systems and before the closing entry.
	Last
	// Closing runs after everything else in the phase. At most one per phase.
	Closing
)

// SystemSpec declares one system registration.
type SystemSpec struct {
	// Name identifies the system in errors. Non-empty and unique within its
	// phase; the schedule rejects duplicates.
	Name string
	// Phase is the schedule the system registers in, bounded like
	// [core.PhasePreStartup]..[core.PhasePostFixed].
	Phase core.Phase
	// Slot selects the phase boundary the system runs at. The zero value is
	// [Regular].
	Slot Slot
	// New constructs the system. Must not be nil.
	New func() core.System
	// Group assigns the system to a named group. Empty means no group.
	Group string
}

// Registration declares one subsystem's wiring: resources, systems, and
// blueprint entries. The root package applies registrations in the order it
// lists them.
type Registration struct {
	// Name identifies the registration in errors. Non-empty.
	Name string
	// Resources provide world resources, applied in order.
	Resources []func(*core.World) error
	// Systems declares the registration's systems, applied in order.
	Systems []SystemSpec
	// Blueprints registers the package's data types with the serialization
	// registry. Nil means none.
	Blueprints func(*serialization.Registry) error
}

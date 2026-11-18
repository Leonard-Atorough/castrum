package collision

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/engine"
)

// Wiring returns the engine registration for the collision subsystem: the
// [Queries] resource and the collision system at the start of the fixed
// phase, before gameplay systems.
func Wiring() engine.Registration {
	return engine.Registration{
		Name: "collision",
		Resources: []func(*core.World) error{
			func(w *core.World) error {
				return w.ProvideEager(func(*core.World) (*Queries, error) { return NewQueries() })
			},
		},
		Systems: []engine.SystemSpec{
			{Name: SystemName, Phase: core.PhaseFixed, Slot: engine.First, New: func() core.System { return NewSystem() }},
		},
	}
}

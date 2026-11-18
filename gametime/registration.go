package gametime

import "github.com/Leonard-Atorough/castrum/serialization"

// Save keys are stable wire identities; renaming one is a breaking change.
const (
	SimulationSaveKey = "gametime.Simulation"
	RealSaveKey       = "gametime.Real"
	GroupsSaveKey     = "gametime.Groups"
)

// RegisterBlueprints registers the package's persistent state with reg:
// the simulation and real-time resources and the timer groups. All three
// are value-shaped - Simulation and Real through their JSON tags, Groups
// through its own JSON projection.
func RegisterBlueprints(reg *serialization.Registry) error {
	if err := reg.Resource[Simulation](SimulationSaveKey); err != nil {
		return err
	}
	if err := reg.Resource[Real](RealSaveKey); err != nil {
		return err
	}
	if err := reg.Resource[Groups](GroupsSaveKey); err != nil {
		return err
	}
	return nil
}

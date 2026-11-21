package gametime

import "github.com/Leonard-Atorough/castrum/serialization"

// Save keys are stable wire identities; renaming one is a breaking change.
const (
	// SimulationSaveKey identifies [Simulation] in save data.
	SimulationSaveKey = "gametime.Simulation"
	// RealSaveKey identifies [Real] in save data.
	RealSaveKey = "gametime.Real"
	// GroupsSaveKey identifies [Groups] in save data.
	GroupsSaveKey = "gametime.Groups"
)

// RegisterBlueprints registers the package's persistent state with reg:
// the [Simulation], [Real], and [Groups] resources.
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

package systems

import (
	"math"

	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
)

type PulseSystem struct {
	pulseQuery *castrum.Query
}

func (ps *PulseSystem) Update(world *castrum.World, delta float64) error {
	for _, entityID := range ps.pulseQuery.EntityIDs() {
		pulse, _ := world.GetComponent[gamecomponents.Pulse](entityID)
		pulse.ElapsedTime += delta // ← Accumulate instead of calling time.Now()

		phase := pulse.ElapsedTime*pulse.Frequency + pulse.TimeOffset
		scaleFactor := 1.0 + pulse.Amplitude*math.Sin(phase)

		transform, _ := world.GetComponent[components.Transform](entityID)
		transform.Scale.X = pulse.StartScale.X * scaleFactor
		transform.Scale.Y = pulse.StartScale.Y * scaleFactor

		world.SetComponent(entityID, transform)
		world.SetComponent(entityID, pulse) // Write back updated ElapsedTime
	}
	return nil
}

func (ps *PulseSystem) Init(world *castrum.World) error {
	ps.pulseQuery = world.NewQuery().WithRequiredComponents(gamecomponents.Pulse{})
	return nil
}

func (ps *PulseSystem) Shutdown(world *castrum.World) error {
	return nil
}

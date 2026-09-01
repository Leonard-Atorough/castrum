package systems

import (
	"math"

	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
)

type PulseSystem struct {
}

func (ps *PulseSystem) Update(world *castrum.World, delta float64) error {
	for _, entityID := range castrum.QueryFor[gamecomponents.Pulse](world) {
		pulse, _ := castrum.GetComponent[gamecomponents.Pulse](world, entityID)
		pulse.ElapsedTime += delta // ← Accumulate instead of calling time.Now()

		phase := pulse.ElapsedTime*pulse.Frequency + pulse.TimeOffset
		scaleFactor := 1.0 + pulse.Amplitude*math.Sin(phase)

		transform, _ := castrum.GetComponent[components.Transform](world, entityID)
		transform.Scale.X = pulse.StartScale.X * scaleFactor
		transform.Scale.Y = pulse.StartScale.Y * scaleFactor

		castrum.SetComponent(world, entityID, transform)
		castrum.SetComponent(world, entityID, pulse) // Write back updated ElapsedTime
	}
	return nil
}

func (ps *PulseSystem) Init(world *castrum.World) error {
	return nil
}

func (ps *PulseSystem) Shutdown(world *castrum.World) error {
	return nil
}

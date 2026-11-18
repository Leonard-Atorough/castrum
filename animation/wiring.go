package animation

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/engine"
)

// Wiring returns the engine registration for the animation subsystem: the
// sprite animation system at the start of the fixed phase.
func Wiring() engine.Registration {
	return engine.Registration{
		Name: "animation",
		Systems: []engine.SystemSpec{
			{Name: SpriteAnimationSystem, Phase: core.PhaseFixed, Slot: engine.First, New: func() core.System { return NewSpriteAnimationSystem() }},
		},
	}
}

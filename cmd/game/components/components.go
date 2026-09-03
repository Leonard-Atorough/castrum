package components

import (
	"github.com/leonard-atorough/castrum/geom"
)

// Velocity represents linear motion in units per second.
type Velocity struct {
	Linear geom.Vector2
}

// Player is a marker component for entities controlled by player input.
type Player struct{}

// Pulse animates an entity's scale by modulating it sinusoidally.
type Pulse struct {
	Frequency   float64      // Hz, how fast to pulse
	Amplitude   float64      // 0-1, how much to scale
	StartScale  geom.Vector2 // base scale to pulse from
	TimeOffset  float64      // seconds, phase offset for the pulse, this allows rolling the animation in time
	ElapsedTime float64      // seconds, keeps track of the elapsed time for the pulse
}

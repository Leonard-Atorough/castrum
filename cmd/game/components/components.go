package components

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/types"
)

// Velocity represents linear motion in units per second.
type Velocity struct {
	Linear types.Vector2
}

// Player is a marker component for entities controlled by player input.
type Player struct{}

// Pulse animates an entity's scale by modulating it sinusoidally.
type Pulse struct {
	Frequency   float64       // Hz, how fast to pulse
	Amplitude   float64       // 0-1, how much to scale
	StartScale  types.Vector2 // base scale to pulse from
	TimeOffset  float64       // seconds, phase offset for the pulse, this allows rolling the animation in time
	ElapsedTime float64       // seconds, keeps track of the elapsed time for the pulse
}

// colliders can have different shapes, so we need a way to support multiple types.
type Collider interface {
	Kind() components.PrimitiveKind
	Shape() any
}

type BoxCollider struct {
	shape types.Rect
}

func (c BoxCollider) Kind() components.PrimitiveKind {
	return components.PrimitiveKindRectangle
}

func (c BoxCollider) Shape() any {
	return c.shape
}

type CircleCollider struct {
	shape types.Circle
}

func (c CircleCollider) Kind() components.PrimitiveKind {
	return components.PrimitiveKindCircle
}

func (c CircleCollider) Shape() any {
	return c.shape
}

// Package physics defines collision event and result data types shared between
// game code and the engine's collision system. The system lives in internal/physicssystem.
package physics

import (
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

type CollisionEventType int

const (
	CollisionEnter CollisionEventType = iota
	CollisionStay
	CollisionExit
)

type PairKey struct {
	EntityA, EntityB ecs.EntityID
}

type CollisionEvent struct {
	CollisionEventType
	PairKey
	Point  geom.Vector2
	Normal geom.Vector2
}

type CollisionResult struct {
	Collided    bool
	Point       geom.Vector2
	Normal      geom.Vector2
	Penetration float64
}

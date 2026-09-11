// Package physics defines collision event and result data types shared between
// game code and the engine's collision system. The system lives in internal/physicssystem.
package physics

import (
	"fmt"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
)

type spatialIndexer interface {
	Query(position geom.Vector2, radius float64) []ecs.EntityID
	Update(entityID ecs.EntityID, position geom.Vector2) error
	Remove(entityID ecs.EntityID)
}

// QueryConfig controls collision detection behavior
type PhysicsConfig struct {
	// CellSize is the size of each cell in the spatial index
	CellSize float64
	// QueryRadius is the default radius for spatial queries
	QueryRadius float64
	// Enabled toggles collision detection on/off
	Enabled bool
}

type PhysicsSystem struct {
	index         spatialIndexer
	config        PhysicsConfig
	previousPairs map[PairKey]*CollisionState
	lastPositions map[ecs.EntityID]geom.Vector2
	dirty         map[ecs.EntityID]struct{}
	query         *ecs.Query
}

func NewSystem(cfg PhysicsConfig) *PhysicsSystem {
	var cellSize = cfg.CellSize
	if cellSize <= 0 {
		cellSize = 50.0 // sensible default
	}
	index, err := newIndex(cellSize)
	if err != nil {
		panic(fmt.Sprintf("failed to create spatial index: %v", err))
	}
	return &PhysicsSystem{
		index:         index,
		config:        cfg,
		previousPairs: make(map[PairKey]*CollisionState),
		lastPositions: make(map[ecs.EntityID]geom.Vector2),
		dirty:         make(map[ecs.EntityID]struct{}),
	}
}

func (s *PhysicsSystem) Init(world *ecs.World) error {
	s.query = world.NewQuery().WithRequiredComponents(
		components.Collider{},
		components.Transform{},
	)
	return nil
}

func (s *PhysicsSystem) Update(world *ecs.World, deltaTime float64) error {
	candidates, err := s.syncIndex(world)
	if err != nil {
		return err
	}

	bus, ok := world.GetResource[*events.EventBus]()
	if !ok {
		return fmt.Errorf("event bus not found")
	}
	collisionErrs := s.narrowphase(world, candidates, bus)

	clear(s.dirty)
	if collisionErrs != nil && !collisionErrs.IsEmpty() {
		return fmt.Errorf("collision errors: %v", collisionErrs)
	}
	return nil
}

func (s *PhysicsSystem) Shutdown(world *ecs.World) error {
	return nil
}

// query helpers for game systems (read-only, after Update ran)
func (s *PhysicsSystem) TestCollision(world *ecs.World, entityA, entityB ecs.EntityID) (CollisionResult, error) {
	colliderA, shapeA, err := s.worldShape(world, entityA)
	if err != nil {
		return CollisionResult{}, err
	}

	colliderB, shapeB, err := s.worldShape(world, entityB)
	if err != nil {
		return CollisionResult{}, err
	}

	if !colliderA.CanCollideWith(&colliderB) {
		return CollisionResult{}, nil
	}

	return intersectsAny(shapeA, shapeB), nil
}
func (s *PhysicsSystem) CollidingWith(world *ecs.World, entityID ecs.EntityID) ([]ecs.EntityID, error) {
	collider, shapeA, err := s.worldShape(world, entityID)
	if err != nil {
		return nil, err
	}

	transform, err := world.GetComponent[components.Transform](entityID)
	if err != nil {
		return nil, fmt.Errorf("entity %d: %w", entityID, err)
	}
	nearby := s.index.Query(transform.Position, s.config.QueryRadius)

	var collisions []ecs.EntityID
	for _, otherID := range nearby {
		if otherID == entityID {
			continue
		}

		other, shapeB, err := s.worldShape(world, otherID)
		if err != nil {
			continue
		}

		if !collider.CanCollideWith(&other) {
			continue
		}

		if intersectsAny(shapeA, shapeB).Collided {
			collisions = append(collisions, otherID)
		}
	}

	return collisions, nil
}

// internal — one pass, one query, three stages
func (s *PhysicsSystem) syncIndex(world *ecs.World) ([]PairKey, error) {
	seen := make(map[ecs.EntityID]struct{}, len(s.lastPositions))

	// One pass: index sync + dirty marking + liveness tracking.
	for result := range s.query.Execute() {
		entityID := result.EntityID

		collider, err := result.Get[components.Collider]()
		if err != nil || !collider.Active {
			continue
		}
		transform, err := result.Get[components.Transform]()
		if err != nil {
			continue
		}

		if err := s.index.Update(entityID, transform.Position); err != nil {
			return nil, err
		}
		seen[entityID] = struct{}{}

		lastPos, exists := s.lastPositions[entityID]
		if !exists || lastPos != transform.Position {
			s.dirty[entityID] = struct{}{}
			s.lastPositions[entityID] = transform.Position
		}
	}

	// Orphans: entity removed, or its Collider went inactive.
	// The index must be told, or stale entries pollute queries forever.
	for entityID := range s.lastPositions {
		if _, ok := seen[entityID]; !ok {
			s.index.Remove(entityID)
			delete(s.lastPositions, entityID)
		}
	}
	for entityID := range s.dirty {
		if _, ok := seen[entityID]; !ok {
			delete(s.dirty, entityID)
		}
	}

	// Drop pair-cache entries for pairs whose members vanished.
	for pair := range s.previousPairs {
		_, okA := seen[pair.EntityA]
		_, okB := seen[pair.EntityB]
		if !okA || !okB {
			delete(s.previousPairs, pair)
		}
	}

	// Broadphase, fused: query neighbors around each dirty entity only.
	candidates := make([]PairKey, 0, len(s.dirty))
	tested := make(map[PairKey]struct{}, len(s.dirty))

	for entityID := range s.dirty {
		transform, err := world.GetComponent[components.Transform](entityID)
		if err != nil {
			continue
		}
		for _, neighborID := range s.index.Query(transform.Position, s.config.QueryRadius) {
			if neighborID == entityID {
				continue
			}
			pair := canonicalPair(entityID, neighborID)
			if _, ok := tested[pair]; ok {
				continue
			}
			tested[pair] = struct{}{}
			candidates = append(candidates, pair)
		}
	}

	return candidates, nil
}

func (s *PhysicsSystem) narrowphase(world *ecs.World, candidates []PairKey, bus *events.EventBus) *CollisionErrors {
	tested := make(map[PairKey]struct{}, len(candidates))
	collisionErrors := &CollisionErrors{}

	// If no candidates and no previous pairs, return early
	if len(candidates) == 0 && len(s.previousPairs) == 0 {
		return collisionErrors
	}

	for _, pair := range candidates {
		tested[pair] = struct{}{}

		result, err := s.TestCollision(world, pair.EntityA, pair.EntityB)
		if err != nil {
			// Accumulate error without logging or deleting pair state.
			// Errors during collision testing (e.g., missing components, invalid shapes) are
			// transient. Preserving pair state allows recovery on the next frame.
			collisionErrors.Add(fmt.Errorf("pair (EntityA=%d, EntityB=%d): %w", pair.EntityA, pair.EntityB, err))
			continue
		}

		prev, wasColliding := s.previousPairs[pair]
		switch {
		case result.Collided && (!wasColliding || !prev.WasColliding):
			s.emit(bus, CollisionEnter, pair, result)
		case result.Collided:
			s.emit(bus, CollisionStay, pair, result)
		case wasColliding && prev.WasColliding:
			s.emit(bus, CollisionExit, pair, prev.CollisionResult)
		}

		if result.Collided {
			s.previousPairs[pair] = &CollisionState{CollisionResult: result, WasColliding: true}
		} else {
			delete(s.previousPairs, pair)
		}
	}

	// Pairs untouched this frame - neither member moved, so the cached result
	// still holds. Replay Stay without re-testing geometry.
	for pair, state := range s.previousPairs {
		if _, ok := tested[pair]; ok {
			continue
		}

		s.emit(bus, CollisionStay, pair, state.CollisionResult)
	}

	return collisionErrors
}

func (s *PhysicsSystem) emit(bus *events.EventBus, eventType CollisionEventType, pair PairKey, result CollisionResult) {
	if bus == nil {
		return // EventBus not registered, skip event emission
	}
	bus.Emit(CollisionEvent{
		CollisionEventType: eventType,
		PairKey:            pair,
		Point:              result.Point,
		Normal:             result.Normal,
	}, "CollisionSystem")
}

func (s *PhysicsSystem) worldShape(world *ecs.World, entityID ecs.EntityID) (components.Collider, any, error) {
	collider, err := world.GetComponent[components.Collider](entityID)
	if err != nil {
		return components.Collider{}, nil, fmt.Errorf("entity %d: %w", entityID, err)
	}

	if !collider.Active {
		return components.Collider{}, nil, fmt.Errorf("collider inactive")
	}

	transform, err := world.GetComponent[components.Transform](entityID)
	if err != nil {
		return components.Collider{}, nil, fmt.Errorf("entity %d: %w", entityID, err)
	}

	return collider, toWorldSpace(collider.Shape, transform.Position), nil
}

// canonicalPair orders a pair so (a, b) and (b, a) collapse to one key.
func canonicalPair(a, b ecs.EntityID) PairKey {
	if a < b {
		return PairKey{EntityA: a, EntityB: b}
	}
	return PairKey{EntityA: b, EntityB: a}
}

// DefaultConfig returns a sensible default collision configuration
func DefaultConfig() PhysicsConfig {
	return PhysicsConfig{
		QueryRadius: 300.0,
		CellSize:    50.0, // default cell size for the spatial index
		Enabled:     true,
	}
}

type PairKey struct {
	EntityA, EntityB ecs.EntityID
}

type CollisionEventType int

const (
	CollisionEnter CollisionEventType = iota
	CollisionStay
	CollisionExit
)

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

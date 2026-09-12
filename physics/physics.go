// Package physics provides broad-phase indexing, narrow-phase collision tests,
// and collision lifecycle events for ECS entities.
package physics

import (
	"fmt"
	"reflect"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/spatial"
)

type spatialIndexer interface {
	QueryInto(bounds geom.Rect, ids []ecs.EntityID) []ecs.EntityID
	Update(entityID ecs.EntityID, bounds geom.Rect) error
	Remove(entityID ecs.EntityID)
}

// PhysicsConfig controls collision detection behavior.
type PhysicsConfig struct {
	// CellSize is the size of each spatial-index cell.
	CellSize float64
	// Enabled toggles collision detection on and off.
	Enabled bool
}

// PhysicsSystem synchronizes active colliders with a bounds-based spatial
// index, tests candidate pairs, and emits collision lifecycle events.
type PhysicsSystem struct {
	index         spatialIndexer
	config        PhysicsConfig
	previousPairs map[PairKey]*CollisionState
	lastProxies   map[ecs.EntityID]collisionProxy
	dirty         map[ecs.EntityID]struct{}
	seen          map[ecs.EntityID]struct{}
	candidates    []PairKey
	tested        map[PairKey]struct{}
	nearby        []ecs.EntityID
	query         *ecs.Query
}

type collisionProxy struct {
	transform components.Transform
	shape     any
	bounds    geom.Rect
}

// NewSystem creates a physics system using cfg. A non-positive cell size uses
// the default size of 50 world units.
func NewSystem(cfg PhysicsConfig) *PhysicsSystem {
	var cellSize = cfg.CellSize
	if cellSize <= 0 {
		cellSize = 50.0 // sensible default
	}
	index, err := spatial.NewGrid(cellSize)
	if err != nil {
		panic(fmt.Sprintf("failed to create spatial index: %v", err))
	}
	return &PhysicsSystem{
		index:         index,
		config:        cfg,
		previousPairs: make(map[PairKey]*CollisionState),
		lastProxies:   make(map[ecs.EntityID]collisionProxy),
		dirty:         make(map[ecs.EntityID]struct{}),
		seen:          make(map[ecs.EntityID]struct{}),
		tested:        make(map[PairKey]struct{}),
		nearby:        make([]ecs.EntityID, 0, 16),
	}
}

// Init prepares the ECS query used to find active collider and transform pairs.
func (s *PhysicsSystem) Init(world *ecs.World) error {
	s.query = world.NewQuery().WithRequiredComponents(
		components.Collider{},
		components.Transform{},
	)
	return nil
}

// Update synchronizes collider proxies and processes collision events for one
// simulation tick. When collision processing is disabled, it performs no
// indexing, narrow-phase work, or event emission.
func (s *PhysicsSystem) Update(world *ecs.World, deltaTime float64) error {
	if !s.config.Enabled {
		return nil
	}

	candidates, err := s.syncIndex()
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

// Shutdown releases no external resources and is provided to satisfy the ECS
// system contract.
func (s *PhysicsSystem) Shutdown(world *ecs.World) error {
	return nil
}

// TestCollision performs an immediate exact collision test for two entities.
// It does not update pair state or emit events.
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

	return intersectsAny(shapeA.shape, shapeB.shape), nil
}

// CollidingWith returns the entities whose exact transformed shapes collide
// with entityID. The spatial index supplies conservative candidates; the
// returned slice contains only narrow-phase hits.
func (s *PhysicsSystem) CollidingWith(world *ecs.World, entityID ecs.EntityID) ([]ecs.EntityID, error) {
	collider, shapeA, err := s.worldShape(world, entityID)
	if err != nil {
		return nil, err
	}

	s.nearby = s.index.QueryInto(shapeA.bounds, s.nearby[:0])

	var collisions []ecs.EntityID
	for _, otherID := range s.nearby {
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

		if intersectsAny(shapeA.shape, shapeB.shape).Collided {
			collisions = append(collisions, otherID)
		}
	}

	return collisions, nil
}

// syncIndex updates changed proxies, removes inactive or deleted entities, and
// builds candidate pairs for dirty proxies and previously colliding pairs.
func (s *PhysicsSystem) syncIndex() ([]PairKey, error) {
	clear(s.seen)
	s.candidates = s.candidates[:0]
	clear(s.tested)

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

		shape, err := transformedCollider(collider.Shape, transform)
		if err != nil {
			return nil, fmt.Errorf("entity %d: %w", entityID, err)
		}
		s.seen[entityID] = struct{}{}

		proxy := collisionProxy{transform: transform, shape: shape.shape, bounds: shape.bounds}
		lastProxy, exists := s.lastProxies[entityID]
		if !exists || collisionProxyChanged(lastProxy, proxy) {
			if err := s.index.Update(entityID, proxy.bounds); err != nil {
				return nil, err
			}
			s.dirty[entityID] = struct{}{}
			s.lastProxies[entityID] = proxy
		}
	}

	// An entity absent from this query was deleted or became inactive. Remove it
	// from every owner of proxy state so stale cells cannot produce candidates.
	for entityID := range s.lastProxies {
		if _, ok := s.seen[entityID]; !ok {
			s.index.Remove(entityID)
			delete(s.lastProxies, entityID)
			delete(s.dirty, entityID)
		}
	}

	// Pair state cannot survive the removal of either member.
	for pair := range s.previousPairs {
		_, okA := s.seen[pair.EntityA]
		_, okB := s.seen[pair.EntityB]
		if !okA || !okB {
			delete(s.previousPairs, pair)
		}
	}

	for entityID := range s.dirty {
		proxy, ok := s.lastProxies[entityID]
		if !ok {
			continue
		}
		s.nearby = s.index.QueryInto(proxy.bounds, s.nearby[:0])
		for _, neighborID := range s.nearby {
			if neighborID == entityID {
				continue
			}
			pair := canonicalPair(entityID, neighborID)
			if _, ok := s.tested[pair]; ok {
				continue
			}
			s.tested[pair] = struct{}{}
			s.candidates = append(s.candidates, pair)
		}
	}
	for pair := range s.previousPairs {
		if _, dirtyA := s.dirty[pair.EntityA]; !dirtyA {
			if _, dirtyB := s.dirty[pair.EntityB]; !dirtyB {
				continue
			}
		}
		if _, tested := s.tested[pair]; tested {
			continue
		}
		s.tested[pair] = struct{}{}
		s.candidates = append(s.candidates, pair)
	}

	return s.candidates, nil
}

func collisionProxyChanged(previous, current collisionProxy) bool {
	return previous.transform.Position != current.transform.Position ||
		previous.transform.Rotation != current.transform.Rotation ||
		previous.transform.Scale != current.transform.Scale ||
		!reflect.DeepEqual(previous.shape, current.shape) ||
		previous.bounds != current.bounds
}

func (s *PhysicsSystem) narrowphase(world *ecs.World, candidates []PairKey, bus *events.EventBus) *CollisionErrors {
	clear(s.tested)
	collisionErrors := &CollisionErrors{}

	// If no candidates and no previous pairs, there is no work to perform.
	if len(candidates) == 0 && len(s.previousPairs) == 0 {
		return collisionErrors
	}

	for _, pair := range candidates {
		s.tested[pair] = struct{}{}

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

	// Untouched pairs have unchanged proxies, so replay Stay from the cached
	// result without repeating the exact geometry test.
	for pair, state := range s.previousPairs {
		if _, ok := s.tested[pair]; ok {
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

func (s *PhysicsSystem) worldShape(world *ecs.World, entityID ecs.EntityID) (components.Collider, transformedShape, error) {
	collider, err := world.GetComponent[components.Collider](entityID)
	if err != nil {
		return components.Collider{}, transformedShape{}, fmt.Errorf("entity %d: %w", entityID, err)
	}

	if !collider.Active {
		return components.Collider{}, transformedShape{}, fmt.Errorf("collider inactive")
	}

	transform, err := world.GetComponent[components.Transform](entityID)
	if err != nil {
		return components.Collider{}, transformedShape{}, fmt.Errorf("entity %d: %w", entityID, err)
	}

	shape, err := transformedCollider(collider.Shape, transform)
	if err != nil {
		return components.Collider{}, transformedShape{}, fmt.Errorf("entity %d: %w", entityID, err)
	}
	return collider, shape, nil
}

// canonicalPair orders a pair so (a, b) and (b, a) collapse to one key.
func canonicalPair(a, b ecs.EntityID) PairKey {
	if a < b {
		return PairKey{EntityA: a, EntityB: b}
	}
	return PairKey{EntityA: b, EntityB: a}
}

// DefaultConfig returns a collision configuration suitable for a typical game.
func DefaultConfig() PhysicsConfig {
	return PhysicsConfig{
		CellSize: 50.0, // default cell size for the spatial index
		Enabled:  true,
	}
}

// PairKey identifies an unordered pair of entities involved in collision
// processing. Use canonicalPair when constructing one from two IDs.
type PairKey struct {
	EntityA, EntityB ecs.EntityID
}

// CollisionEventType identifies a collision lifecycle transition.
type CollisionEventType int

const (
	// CollisionEnter is emitted when a pair starts colliding.
	CollisionEnter CollisionEventType = iota
	// CollisionStay is emitted while a pair remains colliding.
	CollisionStay
	// CollisionExit is emitted when a previously colliding pair separates.
	CollisionExit
)

// CollisionEvent describes a collision lifecycle transition and its contact
// information at the time of the transition.
type CollisionEvent struct {
	CollisionEventType
	PairKey
	Point  geom.Vector2
	Normal geom.Vector2
}

// CollisionResult contains the exact narrow-phase result for a pair.
type CollisionResult struct {
	Collided    bool
	Point       geom.Vector2
	Normal      geom.Vector2
	Penetration float64
}

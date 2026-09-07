// Package physics provides collision detection and resolution for entities
// with Collider components using spatial indexing for efficient queries.
package physics

import (
	"fmt"
	"math"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

type CollisionEventType int

const (
	CollisionEnter CollisionEventType = iota
	CollisionStay
	CollisionExit
)

type PairKey struct {
	EntityA, EntityB core.EntityID
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

type CollisionState struct {
	CollisionResult
	WasColliding bool
}

type spatialIndexer interface {
	Query(position geom.Vector2, radius float64) []core.EntityID
}

type System struct {
	spatial       spatialIndexer
	config        Config
	events        []CollisionEvent
	previousPairs map[PairKey]*CollisionState
	lastPositions map[core.EntityID]geom.Vector2
	dirty         map[core.EntityID]struct{}
	query         *core.Query
}

// Config controls collision detection behavior
type Config struct {
	// QueryRadius is the default radius for spatial queries
	QueryRadius float64
	// Enabled toggles collision detection on/off
	Enabled bool
}

// DefaultConfig returns a sensible default collision configuration
func DefaultConfig() Config {
	return Config{
		QueryRadius: 300.0,
		Enabled:     true,
	}
}

// NewSystem creates a collision system with the given spatial manager
func NewSystem(spatialMgr spatialIndexer, cfg Config) *System {
	return &System{
		spatial:       spatialMgr,
		config:        cfg,
		previousPairs: make(map[PairKey]*CollisionState),
		lastPositions: make(map[core.EntityID]geom.Vector2),
		dirty:         make(map[core.EntityID]struct{}),
	}
}

// Init initializes the collision system
func (s *System) Init(world *core.World) error {
	if s.spatial == nil {
		return fmt.Errorf("collision system requires a spatial indexer")
	}
	s.query = world.NewQuery().WithRequiredComponents(
		components.Collider{},
		components.Transform{},
	)
	return nil
}

func (s *System) Events() []CollisionEvent {
	if s.events == nil {
		return []CollisionEvent{}
	}
	return s.events
}

// Shutdown cleans up the collision system
func (s *System) Shutdown(world *core.World) error {
	return nil
}

// Update rebuilds the collision index for entities that moved since the last
// frame, replays cached results for entities that did not, and emits
// enter/stay/exit events for the frame. It does NOT apply game logic.
func (s *System) Update(world *core.World, deltaTime float64) error {
	if !s.config.Enabled || s.spatial == nil {
		return nil
	}

	s.events = s.events[:0]

	s.markDirtyFromSpatial()

	candidates := s.broadphase(world)
	s.narrowphase(world, candidates)

	clear(s.dirty)
	return nil
}

// TestCollision checks if two entities are colliding at this moment.
// Returns the collision result with contact geometry (point, normal, penetration).
// This is a query helper used by systems to check collisions.
func (s *System) TestCollision(world *core.World, entityA, entityB core.EntityID) (CollisionResult, error) {
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

// QueryCollisions returns all entities colliding with the given entity.
// This is a query helper used by systems to find all current collisions.
func (s *System) QueryCollisions(world *core.World, entityID core.EntityID) ([]core.EntityID, error) {
	collider, shapeA, err := s.worldShape(world, entityID)
	if err != nil {
		return nil, err
	}

	transform, err := world.GetComponent[components.Transform](entityID)
	if err != nil {
		return nil, fmt.Errorf("entity %d: %w", entityID, err)
	}
	nearby := s.spatial.Query(transform.Position, s.config.QueryRadius)

	var collisions []core.EntityID
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

// worldShape fetches an entity's Collider and its shape translated to world
// space. Shared by every code path that needs to test an entity's collider,
// so the fetch-and-translate logic lives in one place.
func (s *System) worldShape(world *core.World, entityID core.EntityID) (components.Collider, any, error) {
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

// markDirtyFromSpatial flags entities whose position changed since the last
// Update call. Any change is enough to mark dirty - a smaller movement can
// still start or end an overlap, so there is no "safe" distance threshold.
func (s *System) markDirtyFromSpatial() {
	seen := make(map[core.EntityID]struct{}, len(s.lastPositions))

	for entry := range s.query.Execute() {
		entityID := entry.EntityID
		seen[entityID] = struct{}{}

		collider, err := entry.Get[components.Collider]()
		if err != nil || !collider.Active {
			continue
		}

		transform, err := entry.Get[components.Transform]()
		if err != nil {
			continue
		}

		lastPos, exists := s.lastPositions[entityID]
		if !exists || lastPos != transform.Position {
			s.dirty[entityID] = struct{}{}
			s.lastPositions[entityID] = transform.Position
		}
	}

	// Cleanup previousPairs entries for entities that no longer exist in the world.
	for pair := range s.previousPairs {
		if _, okA := seen[pair.EntityA]; !okA {
			delete(s.previousPairs, pair)
			continue
		}
		if _, okB := seen[pair.EntityB]; !okB {
			delete(s.previousPairs, pair)
		}
	}

	// Cleanup entries for entities that lost Collider component.
	for entityID := range s.lastPositions {
		if _, ok := seen[entityID]; !ok {
			delete(s.lastPositions, entityID)
			delete(s.dirty, entityID)
		}
	}
}

// broadphase returns candidate pairs by querying the spatial index around
// every dirty entity. Pairs are canonicalized (EntityA < EntityB) and
// deduplicated so a pair moved by both members is only tested once.
func (s *System) broadphase(world *core.World) []PairKey {
	seen := make(map[PairKey]struct{}, len(s.dirty))
	candidates := make([]PairKey, 0, len(s.dirty))

	for entityID := range s.dirty {
		transform, _ := world.GetComponent[components.Transform](entityID)

		for _, neighborID := range s.spatial.Query(transform.Position, s.config.QueryRadius) {
			if neighborID == entityID {
				continue
			}

			pair := canonicalPair(entityID, neighborID)
			if _, ok := seen[pair]; ok {
				continue
			}
			seen[pair] = struct{}{}
			candidates = append(candidates, pair)
		}
	}

	return candidates
}

// narrowphase tests each candidate pair, replays cached Stay events for pairs
// that were colliding but untouched this frame, and emits Enter/Stay/Exit
// events for every pair whose state changed or persists.
func (s *System) narrowphase(world *core.World, candidates []PairKey) {
	tested := make(map[PairKey]struct{}, len(candidates))

	for _, pair := range candidates {
		tested[pair] = struct{}{}

		result, err := s.TestCollision(world, pair.EntityA, pair.EntityB)
		if err != nil {
			delete(s.previousPairs, pair)
			continue
		}

		prev, wasColliding := s.previousPairs[pair]
		switch {
		case result.Collided && (!wasColliding || !prev.WasColliding):
			s.emit(CollisionEnter, pair, result)
		case result.Collided:
			s.emit(CollisionStay, pair, result)
		case wasColliding && prev.WasColliding:
			s.emit(CollisionExit, pair, prev.CollisionResult)
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

		s.emit(CollisionStay, pair, state.CollisionResult)
	}
}

func (s *System) emit(eventType CollisionEventType, pair PairKey, result CollisionResult) {
	s.events = append(s.events, CollisionEvent{
		CollisionEventType: eventType,
		PairKey:            pair,
		Point:              result.Point,
		Normal:             result.Normal,
	})
}

// canonicalPair orders a pair so (a, b) and (b, a) collapse to one key.
func canonicalPair(a, b core.EntityID) PairKey {
	if a < b {
		return PairKey{EntityA: a, EntityB: b}
	}
	return PairKey{EntityA: b, EntityB: a}
}

// toWorldSpace translates a collider shape to world space using the entity's transform position
func toWorldSpace(shape any, position geom.Vector2) any {
	switch s := shape.(type) {
	case geom.Rect:
		return geom.Rect{
			Min: geom.Vector2{X: s.Min.X + position.X, Y: s.Min.Y + position.Y},
			Max: geom.Vector2{X: s.Max.X + position.X, Y: s.Max.Y + position.Y},
		}
	case geom.Circle:
		return geom.Circle{
			Center: geom.Vector2{X: s.Center.X + position.X, Y: s.Center.Y + position.Y},
			Radius: s.Radius,
		}
	}
	return shape
}

func intersectsAny(shapeA, shapeB any) CollisionResult {
	switch a := shapeA.(type) {
	case geom.Rect:
		switch b := shapeB.(type) {
		case geom.Rect:
			// Rect-Rect contact geometry deferred (SAT solver v0.2.0)
			return CollisionResult{Collided: a.Intersects(b)}
		case geom.Circle:
			return circleRectContact(b, a)
		}
	case geom.Circle:
		switch b := shapeB.(type) {
		case geom.Rect:
			return circleRectContact(a, b)
		case geom.Circle:
			return circleCircleContact(a, b)
		}
	}
	return CollisionResult{}
}

func circleCircleContact(a, b geom.Circle) CollisionResult {
	dx := b.Center.X - a.Center.X
	dy := b.Center.Y - a.Center.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	minDist := a.Radius + b.Radius

	if dist > minDist {
		return CollisionResult{Collided: false}
	}

	if dist == 0 {
		// Circles at same position, arbitrary normal.
		return CollisionResult{
			Collided:    true,
			Penetration: minDist,
			Normal:      geom.Vector2{X: 1, Y: 0},
			Point:       a.Center,
		}
	}

	nx := dx / dist
	ny := dy / dist
	return CollisionResult{
		Collided:    true,
		Penetration: minDist - dist,
		Normal:      geom.Vector2{X: nx, Y: ny},
		Point:       geom.Vector2{X: a.Center.X + nx*a.Radius, Y: a.Center.Y + ny*a.Radius},
	}
}

func circleRectContact(circle geom.Circle, rect geom.Rect) CollisionResult {
	// Closest point on rect to circle center.
	closestX := math.Max(rect.Min.X, math.Min(circle.Center.X, rect.Max.X))
	closestY := math.Max(rect.Min.Y, math.Min(circle.Center.Y, rect.Max.Y))

	dx := circle.Center.X - closestX
	dy := circle.Center.Y - closestY
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > circle.Radius {
		return CollisionResult{Collided: false}
	}

	if dist == 0 {
		// Circle center inside rect, arbitrary normal outward.
		return CollisionResult{
			Collided:    true,
			Penetration: circle.Radius,
			Normal:      geom.Vector2{X: 1, Y: 0},
			Point:       circle.Center,
		}
	}

	nx := dx / dist
	ny := dy / dist
	return CollisionResult{
		Collided:    true,
		Penetration: circle.Radius - dist,
		Normal:      geom.Vector2{X: nx, Y: ny},
		Point:       geom.Vector2{X: closestX, Y: closestY},
	}
}

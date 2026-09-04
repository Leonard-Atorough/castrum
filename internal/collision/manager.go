// Package collision provides collision detection and resolution for entities
// with Collider components using spatial indexing for efficient queries.
package collision

import (
	"fmt"
	"math"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/spatial"
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

type Manager struct {
	spatial       *spatial.Manager
	config        Config
	events        []CollisionEvent
	previousPairs map[PairKey]*CollisionState

	lastPositions map[core.EntityID]geom.Vector2
	dirty         map[core.EntityID]struct{}
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

// NewManager creates a collision manager with the given spatial manager
func NewManager(spatialMgr *spatial.Manager, cfg Config) *Manager {
	return &Manager{
		spatial:       spatialMgr,
		config:        cfg,
		previousPairs: make(map[PairKey]*CollisionState),
		lastPositions: make(map[core.EntityID]geom.Vector2),
		dirty:         make(map[core.EntityID]struct{}),
	}
}

// Init initializes the collision manager
func (m *Manager) Init(world *core.World) error {
	if m.spatial == nil {
		return fmt.Errorf("collision manager requires a spatial manager")
	}
	return nil
}

func (m *Manager) Events() []CollisionEvent {
	return m.events
}

// Update rebuilds the collision index for entities that moved since the last
// frame, replays cached results for entities that did not, and emits
// enter/stay/exit events for the frame. It does NOT apply game logic.
func (m *Manager) Update(world *core.World, deltaTime float64) error {
	if !m.config.Enabled || m.spatial == nil {
		return nil
	}

	m.events = m.events[:0]

	m.markDirtyFromSpatial(world)

	candidates := m.broadphase(world)
	m.narrowphase(world, candidates)

	clear(m.dirty)
	return nil
}

// TestCollision checks if two entities are colliding at this moment.
// Returns the collision result with contact geometry (point, normal, penetration).
func (m *Manager) TestCollision(world *core.World, entityA, entityB core.EntityID) (CollisionResult, error) {
	colliderA, shapeA, err := m.worldShape(world, entityA)
	if err != nil {
		return CollisionResult{}, err
	}

	colliderB, shapeB, err := m.worldShape(world, entityB)
	if err != nil {
		return CollisionResult{}, err
	}

	if !colliderA.CanCollideWith(&colliderB) {
		return CollisionResult{}, nil
	}

	return intersectsAny(shapeA, shapeB), nil
}

// QueryCollisions returns all entities colliding with the given entity
func (m *Manager) QueryCollisions(world *core.World, entityID core.EntityID) ([]core.EntityID, error) {
	collider, shapeA, err := m.worldShape(world, entityID)
	if err != nil {
		return nil, err
	}

	transform, err := core.GetComponent[components.Transform](world, entityID)
	if err != nil {
		return nil, fmt.Errorf("entity %d: %w", entityID, err)
	}
	nearby := m.spatial.GetNearbyEntities(transform.Position, m.config.QueryRadius)

	var collisions []core.EntityID
	for _, otherID := range nearby {
		if otherID == entityID {
			continue
		}

		other, shapeB, err := m.worldShape(world, otherID)
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
func (m *Manager) worldShape(world *core.World, entityID core.EntityID) (components.Collider, any, error) {
	collider, err := core.GetComponent[components.Collider](world, entityID)
	if err != nil {
		return components.Collider{}, nil, fmt.Errorf("entity %d: %w", entityID, err)
	}

	if !collider.Active {
		return components.Collider{}, nil, fmt.Errorf("collider inactive")
	}

	transform, err := core.GetComponent[components.Transform](world, entityID)
	if err != nil {
		return components.Collider{}, nil, fmt.Errorf("entity %d: %w", entityID, err)
	}

	return collider, toWorldSpace(collider.Shape, transform.Position), nil
}

// markDirtyFromSpatial flags entities whose position changed since the last
// Update call. Any change is enough to mark dirty - a smaller movement can
// still start or end an overlap, so there is no "safe" distance threshold.
func (m *Manager) markDirtyFromSpatial(world *core.World) {
	seen := make(map[core.EntityID]struct{}, len(m.lastPositions))

	for entry := range world.NewQuery().WithRequiredComponents(components.Transform{}, components.Collider{}).Execute() {
		entityID := entry.EntityID
		seen[entityID] = struct{}{}

		collider, err := core.GetComponent[components.Collider](world, entityID)
		if err != nil || !collider.Active {
			continue
		}

		transform, err := core.GetComponent[components.Transform](world, entityID)
		if err != nil {
			continue
		}

		lastPos, exists := m.lastPositions[entityID]
		if !exists || lastPos != transform.Position {
			m.dirty[entityID] = struct{}{}
			m.lastPositions[entityID] = transform.Position
		}
	}

	// Cleanup previousPairs entries for entities that no longer exist in the world.
	for pair := range m.previousPairs {
		if _, okA := seen[pair.EntityA]; !okA {
			delete(m.previousPairs, pair)
			continue
		}
		if _, okB := seen[pair.EntityB]; !okB {
			delete(m.previousPairs, pair)
		}
	}

	// Cleanup entries for entities that lost Collider component.
	for entityID := range m.lastPositions {
		if _, ok := seen[entityID]; !ok {
			delete(m.lastPositions, entityID)
			delete(m.dirty, entityID)
		}
	}
}

// broadphase returns candidate pairs by querying the spatial index around
// every dirty entity. Pairs are canonicalized (EntityA < EntityB) and
// deduplicated so a pair moved by both members is only tested once.
func (m *Manager) broadphase(world *core.World) []PairKey {
	seen := make(map[PairKey]struct{}, len(m.dirty))
	candidates := make([]PairKey, 0, len(m.dirty))

	for entityID := range m.dirty {
		transform, _ := core.GetComponent[components.Transform](world, entityID)

		for _, neighborID := range m.spatial.GetNearbyEntities(transform.Position, m.config.QueryRadius) {
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
func (m *Manager) narrowphase(world *core.World, candidates []PairKey) {
	tested := make(map[PairKey]struct{}, len(candidates))

	for _, pair := range candidates {
		tested[pair] = struct{}{}

		result, err := m.TestCollision(world, pair.EntityA, pair.EntityB)
		if err != nil {
			delete(m.previousPairs, pair)
			continue
		}

		prev, wasColliding := m.previousPairs[pair]
		switch {
		case result.Collided && (!wasColliding || !prev.WasColliding):
			m.emit(CollisionEnter, pair, result)
		case result.Collided:
			m.emit(CollisionStay, pair, result)
		case wasColliding && prev.WasColliding:
			m.emit(CollisionExit, pair, prev.CollisionResult)
		}

		if result.Collided {
			m.previousPairs[pair] = &CollisionState{CollisionResult: result, WasColliding: true}
		} else {
			delete(m.previousPairs, pair)
		}
	}

	// Pairs untouched this frame - neither member moved, so the cached result
	// still holds. Replay Stay without re-testing geometry.
	for pair, state := range m.previousPairs {
		if _, ok := tested[pair]; ok {
			continue
		}

		m.emit(CollisionStay, pair, state.CollisionResult)
	}
}

func (m *Manager) emit(eventType CollisionEventType, pair PairKey, result CollisionResult) {
	m.events = append(m.events, CollisionEvent{
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

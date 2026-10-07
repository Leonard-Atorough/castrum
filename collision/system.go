package collision

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/internal/spatial"
)

// cellSize is the broad-phase grid's cell edge in world units. It
// trades scan cost against index size and never affects which pairs
// collide: the grid only supplies candidates, the narrow phase
// decides.
const cellSize = 50.0

// NewSystem returns the collision system: broad-phase indexing,
// narrow-phase testing, and the contact lifecycle written as
// [Contacts] state. Detection only - there is no response and no
// event bus; a game derives enter/stay/exit by comparing a
// collider's Current and Previous contacts.
//
// [castrum.New] registers this system; games never register it
// themselves. It runs early in the fixed phase, ahead of gameplay
// systems, so it reads the transforms as of the end of last tick -
// a movement this tick is detected next tick.
//
// The system attaches Contacts to every entity that carries a
// Collider, transforms active colliders by rotation and position
// (never scale), and re-tests every changed collider against its
// surroundings plus every pair that was already colliding, so a pair
// separating is always observed.
func NewSystem() core.System {
	return &system{
		proxies:     make(map[core.EntityID]proxy),
		dirty:       make(map[core.EntityID]struct{}),
		seen:        make(map[core.EntityID]struct{}),
		oldContacts: make(map[core.EntityID][]Contact),
		newContacts: make(map[core.EntityID][]Contact),
		tested:      make(map[pairKey]struct{}),
		nearby:      make([]core.EntityID, 0, 16),
		attachIDs:   make([]core.EntityID, 0, 16),
	}
}

// proxy is an active collider's world-space snapshot. Two equal
// proxies mean nothing observable changed, so the collider is not
// re-tested against its surroundings this tick.
type proxy struct {
	collider Collider
	shape    worldShape
	bounds   geom.Rect
}

// pairKey identifies an unordered pair of entities; build it with
// [canonicalPair] so (a, b) and (b, a) collapse to one key.
type pairKey struct {
	a, b core.EntityID
}

func canonicalPair(x, y core.EntityID) pairKey {
	if x < y {
		return pairKey{x, y}
	}
	return pairKey{y, x}
}

// system is the collision system's per-world state. It owns the
// broad-phase index and the per-tick scratch; components carry the
// lifecycle.
type system struct {
	ensure *core.Query
	update *core.Query
	grid   *spatial.Grid

	proxies     map[core.EntityID]proxy
	dirty       map[core.EntityID]struct{}
	seen        map[core.EntityID]struct{}
	oldContacts map[core.EntityID][]Contact
	newContacts map[core.EntityID][]Contact
	tested      map[pairKey]struct{}
	candidates  []pairKey
	nearby      []core.EntityID
	attachIDs   []core.EntityID
}

// Update runs one fixed tick of collision detection: attach missing
// Contacts, sync the broad-phase index with changed colliders,
// collect and test candidate pairs, and write each entity's Contacts.
func (s *system) Update(ctx *core.Context) error {
	if s.ensure == nil {
		s.ensure = core.NewQuery(ctx.World).With(Collider{}).Without(Contacts{})
		s.update = core.NewQuery(ctx.World).With(Collider{}, core.Transform{}, Contacts{})
		grid, err := spatial.NewGrid(cellSize)
		if err != nil {
			return fmt.Errorf("collision: create spatial index: %w", err)
		}
		s.grid = grid
	}
	s.resetTick()

	if err := s.attachContacts(ctx.World); err != nil {
		return err
	}
	if err := s.syncProxies(); err != nil {
		return err
	}
	s.collectCandidates()
	s.narrowPhase()
	return s.writeContacts()
}

func (s *system) resetTick() {
	clear(s.dirty)
	clear(s.seen)
	clear(s.oldContacts)
	clear(s.newContacts)
	clear(s.tested)
	s.candidates = s.candidates[:0]
}

// attachContacts gives every collider entity its Contacts component.
// The add happens between passes, when no iteration is active, and
// only the first tick a collider exists costs anything.
func (s *system) attachContacts(world *core.World) error {
	s.attachIDs = s.attachIDs[:0]
	for e := range s.ensure.Execute() {
		s.attachIDs = append(s.attachIDs, e.ID())
	}
	for _, id := range s.attachIDs {
		if err := core.NewEntity(id).AddComponent(world, Contacts{}); err != nil {
			return fmt.Errorf("collision: attach contacts to entity %d: %w", id, err)
		}
	}
	return nil
}

// syncProxies refreshes the world-space snapshot of every active
// collider, updates the index for the changed ones, and drops every
// trace of proxies that vanished - destroyed, deactivated, or
// stripped of their Transform.
func (s *system) syncProxies() error {
	for e := range s.update.Execute() {
		id := e.ID()
		collider, _ := e.Component[Collider]()
		transform, _ := e.Component[core.Transform]()
		contacts, _ := e.Component[Contacts]()
		s.oldContacts[id] = contacts.Current

		if !collider.Active {
			continue
		}
		shape := transformShape(collider.Shape, collider.Offset, transform)
		if shape.shape == nil {
			return fmt.Errorf("collision: entity %d has no usable collider shape", id)
		}
		p := proxy{collider: collider, shape: shape.shape, bounds: shape.bounds}
		if last, ok := s.proxies[id]; !ok || last != p {
			if err := s.grid.Update(id, shape.bounds); err != nil {
				return fmt.Errorf("collision: entity %d: %w", id, err)
			}
			s.proxies[id] = p
			s.dirty[id] = struct{}{}
		}
		s.seen[id] = struct{}{}
	}

	for id := range s.proxies {
		if _, ok := s.seen[id]; !ok {
			s.grid.Remove(id)
			delete(s.proxies, id)
			delete(s.dirty, id)
		}
	}
	return nil
}

// collectCandidates builds this tick's pair list: every changed
// collider against everything near it now, plus every pair that was
// already colliding, so a separating pair is re-tested and its exit
// observed even when neither proxy changed.
func (s *system) collectCandidates() {
	for id := range s.dirty {
		p := s.proxies[id]
		s.nearby = s.grid.QueryInto(p.bounds, s.nearby[:0])
		for _, neighbor := range s.nearby {
			if neighbor == id {
				continue
			}
			s.addCandidate(canonicalPair(id, neighbor))
		}
	}
	for id, contacts := range s.oldContacts {
		for _, c := range contacts {
			s.addCandidate(canonicalPair(id, c.Other))
		}
	}
}

func (s *system) addCandidate(pair pairKey) {
	if _, ok := s.tested[pair]; ok {
		return
	}
	s.tested[pair] = struct{}{}
	s.candidates = append(s.candidates, pair)
}

// narrowPhase runs the exact test for each candidate pair that the
// layer/mask filters allow and records hits for both entities. A
// miss records nothing: absence from Current is the exit signal.
func (s *system) narrowPhase() {
	for _, pair := range s.candidates {
		a, okA := s.proxies[pair.a]
		b, okB := s.proxies[pair.b]
		if !okA || !okB {
			continue
		}
		if !a.collider.CanCollideWith(b.collider) {
			continue
		}
		hit := contactBetween(a.shape, b.shape)
		if !hit.collided {
			continue
		}
		trigger := a.collider.Trigger || b.collider.Trigger
		s.newContacts[pair.a] = append(s.newContacts[pair.a], Contact{
			Other:       pair.b,
			Point:       hit.point,
			Normal:      hit.normal,
			Penetration: hit.penetration,
			Trigger:     trigger,
		})
		s.newContacts[pair.b] = append(s.newContacts[pair.b], Contact{
			Other:       pair.a,
			Point:       hit.point,
			Normal:      hit.normal.Mul(-1),
			Penetration: hit.penetration,
			Trigger:     trigger,
		})
	}
}

// writeContacts rotates each collider's state - last tick's Current
// becomes Previous - and installs this tick's contacts, sorted so
// successive ticks compare stably.
func (s *system) writeContacts() error {
	for id := range s.newContacts {
		contacts := s.newContacts[id]
		slices.SortFunc(contacts, func(x, y Contact) int {
			return cmp.Compare(x.Other, y.Other)
		})
	}

	for e := range s.update.Execute() {
		id := e.ID()
		current := s.newContacts[id]
		previous := s.oldContacts[id]
		e.Update(func(c *Contacts) {
			c.Previous = previous
			c.Current = current
		})
	}
	return nil
}

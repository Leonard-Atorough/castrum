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

// SystemName is the name castrum.New registers the collision system
// under in the fixed schedule.
const SystemName = "engine.collision"

// NewSystem returns the collision detector, which records overlaps in
// [Contacts].
//
// castrum.New registers it under [SystemName] when configured with
// `castrum.WithCollision`; otherwise, register it directly. It runs before
// gameplay systems in the fixed phase, so it reads transforms from the
// preceding tick and detects movement made during the current tick on the
// next one.
//
// The system attaches [Contacts] to entities with a [Collider], applies
// position and rotation but not scale, and retains existing pairs long enough
// to report when a contact ends.
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

// Equal proxies need no broad-phase update or re-test.
type proxy struct {
	collider Collider
	shape    worldShape
	bounds   geom.Rect
}

// pairKey identifies an unordered pair; use [canonicalPair] to keep its order
// consistent.
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
	ensure  *core.Query
	update  *core.Query
	orphans [2]*core.Query
	grid    *spatial.Grid

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

// Update runs one fixed tick of collision detection: reconcile the
// Contacts components with who is a collider, sync the broad-phase
// index with changed colliders, collect and test candidate pairs,
// and write each entity's Contacts.
func (s *system) Update(ctx *core.Context) error {
	if s.ensure == nil {
		s.ensure = core.NewQuery(ctx.World).With(Collider{}).Without(Contacts{})
		s.update = core.NewQuery(ctx.World).With(Collider{}, core.Transform{}, Contacts{})

		s.orphans[0] = core.NewQuery(ctx.World).With(Contacts{}, Collider{}).Without(core.Transform{})
		s.orphans[1] = core.NewQuery(ctx.World).With(Contacts{}).Without(Collider{})
		grid, err := spatial.NewGrid(cellSize)
		if err != nil {
			return fmt.Errorf("collision: create spatial index: %w", err)
		}
		s.grid = grid
	}
	s.resetTick()

	if err := s.reconcileContacts(ctx.World); err != nil {
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

// reconcileContacts removes stale contact state so removing and later
// re-adding a collider cannot preserve an old contact.
func (s *system) reconcileContacts(world *core.World) error {
	s.attachIDs = s.attachIDs[:0]
	for e := range s.ensure.Execute() {
		s.attachIDs = append(s.attachIDs, e.ID())
	}
	for _, id := range s.attachIDs {
		if err := core.NewEntity(id).AddComponent(world, Contacts{}); err != nil {
			return fmt.Errorf("collision: attach contacts to entity %d: %w", id, err)
		}
	}

	for _, orphan := range s.orphans {
		s.attachIDs = s.attachIDs[:0]
		for e := range orphan.Execute() {
			s.attachIDs = append(s.attachIDs, e.ID())
		}
		for _, id := range s.attachIDs {
			if err := core.NewEntity(id).RemoveComponent[Contacts](world); err != nil {
				return fmt.Errorf("collision: detach contacts from entity %d: %w", id, err)
			}
		}
	}
	return nil
}

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

// collectCandidates includes existing pairs so separating colliders are
// retested and their exits are observed even when neither proxy changed.
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

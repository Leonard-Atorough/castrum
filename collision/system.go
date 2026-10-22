package collision

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/internal/spatial"
	"github.com/Leonard-Atorough/castrum/signal"
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
		proxies:   make(map[core.EntityID]*proxy),
		contacts:  make(map[core.EntityID]*contactBuffers),
		tested:    make(map[pairKey]struct{}),
		nearby:    make([]core.EntityID, 0, 16),
		attachIDs: make([]core.EntityID, 0, 16),
	}
}

// A failed sync can cache inputs before their candidates are collected.
type proxy struct {
	id       core.EntityID
	collider Collider
	position geom.Vector2
	rotation float64
	shape    worldShape
	bounds   geom.Rect
	seen     uint64
	dirty    bool
}

// The spare buffer is two ticks old, so building this tick cannot overwrite
// either slice currently exposed through Contacts.
type contactBuffers struct {
	current  []Contact
	previous []Contact
	spare    []Contact
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

	proxies    map[core.EntityID]*proxy
	active     []*proxy
	nextActive []*proxy
	contacts   map[core.EntityID]*contactBuffers
	tick       uint64
	dirty      []core.EntityID
	tested     map[pairKey]struct{}
	candidates []pairKey
	nearby     []core.EntityID
	attachIDs  []core.EntityID
	hub        *signal.Hub
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
	if s.hub == nil {
		hub, err := ctx.World.Resource[*signal.Hub]()
		if err != nil {
			return fmt.Errorf("collision system: resolve signal hub: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
		}
		s.hub = hub
	}
	s.resetTick()

	if err := s.reconcileContacts(ctx.World); err != nil {
		return err
	}
	if err := s.syncProxies(); err != nil {
		// A failed pass may already have indexed new proxies. Keep them in the
		// row cache so a retry can also remove any that have since disappeared.
		s.active = s.active[:0]
		for _, p := range s.proxies {
			s.active = append(s.active, p)
		}
		return err
	}
	s.collectCandidates()
	s.narrowPhase()
	return s.writeContacts(ctx.World, ctx.Tick)
}

func (s *system) resetTick() {
	s.tick++
	s.dirty = s.dirty[:0]
	s.nextActive = s.nextActive[:0]
	for _, buffers := range s.contacts {
		buffers.spare = buffers.spare[:0]
	}
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
		delete(s.contacts, id)
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
			delete(s.contacts, id)
		}
	}
	return nil
}

func (s *system) syncProxies() error {
	retained := 0
	for e := range s.update.Execute() {
		id := e.ID()
		collider, _ := e.Component[Collider]()
		transform, _ := e.Component[core.Transform]()
		if !collider.Active {
			continue
		}
		// Query order is stable until structure changes; verify the ID before
		// reusing the prior row's proxy and fall back to lookup after migration.
		var p *proxy
		if i := len(s.nextActive); i < len(s.active) && s.active[i].id == id {
			p = s.active[i]
		} else {
			p = s.proxies[id]
		}
		exists := p != nil
		if exists {
			retained++
		}
		if exists && p.position == transform.Position && p.rotation == transform.Rotation && p.collider == collider {
			p.seen = s.tick
			s.nextActive = append(s.nextActive, p)
			if p.dirty {
				s.dirty = append(s.dirty, id)
			}
			continue
		}
		if !exists {
			p = &proxy{id: id}
		}
		geometryChanged := !exists || p.position != transform.Position ||
			p.rotation != transform.Rotation || p.collider.Shape != collider.Shape ||
			p.collider.Offset != collider.Offset
		changed := geometryChanged || p.collider != collider
		if geometryChanged {
			shape := transformShape(collider.Shape, collider.Offset, transform)
			if shape.shape == nil {
				return fmt.Errorf("collision: entity %d has no usable collider shape", id)
			}
			if !exists || p.bounds != shape.bounds {
				if err := s.grid.Update(id, shape.bounds); err != nil {
					return fmt.Errorf("collision: entity %d: %w", id, err)
				}
			}
			p.shape, p.bounds = shape.shape, shape.bounds
			p.position, p.rotation = transform.Position, transform.Rotation
		}
		p.collider, p.seen = collider, s.tick
		p.dirty = p.dirty || changed
		if !exists {
			s.proxies[id] = p
		}
		s.nextActive = append(s.nextActive, p)
		if p.dirty {
			s.dirty = append(s.dirty, id)
		}
	}

	if retained != len(s.active) {
		for _, p := range s.active {
			if p.seen != s.tick {
				s.grid.Remove(p.id)
				delete(s.proxies, p.id)
			}
		}
	}
	clear(s.active)
	s.active, s.nextActive = s.nextActive, s.active
	return nil
}

// collectCandidates includes existing pairs so separating colliders are
// retested and their exits are observed even when neither proxy changed.
func (s *system) collectCandidates() {
	for _, id := range s.dirty {
		p := s.proxies[id]
		s.nearby = s.grid.QueryOverlapsInto(p.bounds, s.nearby[:0])
		for _, neighbor := range s.nearby {
			if neighbor == id {
				continue
			}
			other := s.proxies[neighbor]
			if !p.collider.CanCollideWith(other.collider) {
				continue
			}
			s.addCandidate(canonicalPair(id, neighbor))
		}
		p.dirty = false
	}
	for id, buffers := range s.contacts {
		for _, c := range buffers.current {
			if id < c.Other {
				s.addCandidate(pairKey{id, c.Other})
			}
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
		s.appendContact(pair.a, Contact{
			Other:       pair.b,
			Point:       hit.point,
			Normal:      hit.normal,
			Penetration: hit.penetration,
			Trigger:     trigger,
		})
		s.appendContact(pair.b, Contact{
			Other:       pair.a,
			Point:       hit.point,
			Normal:      hit.normal.Mul(-1),
			Penetration: hit.penetration,
			Trigger:     trigger,
		})
	}
}

func (s *system) appendContact(id core.EntityID, c Contact) {
	buffers := s.contacts[id]
	if buffers == nil {
		buffers = &contactBuffers{}
		s.contacts[id] = buffers
	}
	buffers.spare = append(buffers.spare, c)
}

func (s *system) writeContacts(world *core.World, tick uint64) error {
	for id, buffers := range s.contacts {
		entity := core.NewEntity(id)
		if !entity.HasComponent[Contacts](world) {
			delete(s.contacts, id)
			continue
		}
		slices.SortFunc(buffers.spare, func(x, y Contact) int {
			return cmp.Compare(x.Other, y.Other)
		})
		current, previous := buffers.spare, buffers.current
		if slices.Equal(current, buffers.current) && slices.Equal(previous, buffers.previous) {
			continue
		}
		if err := entity.SetComponent(world, Contacts{Current: current, Previous: previous}); err != nil {
			return fmt.Errorf("collision: write contacts to entity %d: %w", id, err)
		}
		s.emitEdges(id, current, previous, tick)
		buffers.spare = buffers.previous
		buffers.previous, buffers.current = previous, current
		if len(current) == 0 && len(previous) == 0 {
			delete(s.contacts, id)
		}
	}
	return nil
}

// emitEdges merges the two Other-sorted slices in one walk, emitting
// Entered and Exited for the entries that match nothing on the other side.
func (s *system) emitEdges(id core.EntityID, current, previous []Contact, tick uint64) {
	i, j := 0, 0
	for i < len(current) && j < len(previous) {
		switch {
		case current[i].Other < previous[j].Other:
			s.hub.Emit(enteredSignal(id, current[i], tick))
			i++
		case current[i].Other > previous[j].Other:
			s.hub.Emit(Exited{Entity: id, Other: previous[j].Other, Tick: tick})
			j++
		default:
			i++
			j++
		}
	}
	for ; i < len(current); i++ {
		s.hub.Emit(enteredSignal(id, current[i], tick))
	}
	for ; j < len(previous); j++ {
		s.hub.Emit(Exited{Entity: id, Other: previous[j].Other, Tick: tick})
	}
}

func enteredSignal(id core.EntityID, c Contact, tick uint64) Entered {
	return Entered{
		Entity:      id,
		Other:       c.Other,
		Point:       c.Point,
		Normal:      c.Normal,
		Penetration: c.Penetration,
		Trigger:     c.Trigger,
		Tick:        tick,
	}
}

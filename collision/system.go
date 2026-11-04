package collision

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/core"
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

// The spare buffer is two ticks old, so building this tick cannot overwrite
// either slice currently exposed through Contacts.
type contactBuffers struct {
	current  []Contact
	previous []Contact
	spare    []Contact
}

type pairKey struct {
	a, b core.EntityID
}

func canonicalPair(x, y core.EntityID) pairKey {
	if x < y {
		return pairKey{x, y}
	}
	return pairKey{y, x}
}

// NewSystem returns the collision detector, which records overlaps in
// [Contacts].
//
// castrum.New registers it under [SystemName] when configured with
// `castrum.WithCollision`; otherwise, register it directly and provide
// [Queries] and a [signal.Hub] to the world. It runs before gameplay systems
// in the fixed phase, so it reads transforms from the preceding tick and
// detects movement made during the current tick on the next one.
//
// The system attaches [Contacts] to entities with a [Collider], applies
// position and rotation but not scale, and retains existing pairs long enough
// to report when a contact ends.
func NewSystem() core.System {
	return &system{
		contacts:  make(map[core.EntityID]*contactBuffers),
		tested:    make(map[pairKey]struct{}),
		nearby:    make([]core.EntityID, 0, 16),
		attachIDs: make([]core.EntityID, 0, 16),
	}
}

// system owns contact detection state. The [Queries] resource owns the
// broadphase shared with on-demand spatial queries.
type system struct {
	ensure  *core.Query
	orphans [2]*core.Query

	queries    *Queries
	contacts   map[core.EntityID]*contactBuffers
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
		s.orphans[0] = core.NewQuery(ctx.World).With(Contacts{}, Collider{}).Without(core.Transform{})
		s.orphans[1] = core.NewQuery(ctx.World).With(Contacts{}).Without(Collider{})
	}
	if s.hub == nil {
		hub, err := ctx.World.Resource[*signal.Hub]()
		if err != nil {
			return fmt.Errorf("collision system: resolve signal hub: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
		}
		s.hub = hub
	}
	if s.queries == nil {
		queries, err := ctx.World.Resource[*Queries]()
		if err != nil {
			return fmt.Errorf("collision system: resolve queries: %w; provide *collision.Queries using collision.NewQueries, or construct the game with castrum.WithCollision", err)
		}
		s.queries = queries
	}
	s.resetTick()

	if err := s.reconcileContacts(ctx.World); err != nil {
		return err
	}
	changedIDs, err := s.queries.sync(ctx.World, ctx.Tick)
	if err != nil {
		return err
	}
	s.collectCandidates(changedIDs)
	s.narrowPhase()
	return s.writeContacts(ctx.World, ctx.Tick)
}

func (s *system) resetTick() {
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

func (s *system) collectCandidates(changedIDs []core.EntityID) {
	bp := s.queries.broadphase
	for _, id := range changedIDs {
		p := bp.proxies[id]
		if p == nil {
			continue
		}
		s.nearby = bp.grid.QueryOverlapsInto(p.bounds, s.nearby[:0])
		for _, neighbor := range s.nearby {
			if neighbor == id {
				continue
			}
			other := bp.proxies[neighbor]
			if other == nil || !p.collider.CanCollideWith(other.collider) {
				continue
			}
			s.addCandidate(canonicalPair(id, neighbor))
		}
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
	proxies := s.queries.broadphase.proxies
	for _, pair := range s.candidates {
		a, okA := proxies[pair.a]
		b, okB := proxies[pair.b]
		if !okA || !okB || !a.collider.CanCollideWith(b.collider) {
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

package collision

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/internal/spatial"
)

type proxy struct {
	id       core.EntityID
	collider Collider
	position geom.Vector2
	rotation float64
	shape    worldShape
	bounds   geom.Rect
}

type proxyChange struct {
	id    core.EntityID
	value proxy
}

type broadphase struct {
	grid    *spatial.Grid
	proxies map[core.EntityID]*proxy
	world   *core.World
	query   *core.Query
	seen    map[core.EntityID]struct{}

	updates    []proxyChange
	changes    []spatial.Change
	changedIDs []core.EntityID
	tick       uint64
}

func newBroadphase() (*broadphase, error) {
	grid, err := spatial.NewGrid(cellSize)
	if err != nil {
		return nil, fmt.Errorf("collision: create spatial index: %w", err)
	}
	return &broadphase{
		grid:    grid,
		proxies: make(map[core.EntityID]*proxy),
		seen:    make(map[core.EntityID]struct{}),
	}, nil
}

// Sync stages the complete collider state before changing the index. A
// validation or grid-batch error leaves the published proxies and tick intact.
func (b *broadphase) Sync(world *core.World, tick uint64) ([]core.EntityID, error) {
	if b.world != nil && b.world != world {
		return nil, fmt.Errorf("collision: Queries resource is already bound to another world")
	}
	if b.query == nil {
		b.world = world
		b.query = core.NewQuery(world).With(Collider{}, core.Transform{})
	}

	clear(b.seen)
	b.updates = b.updates[:0]
	b.changes = b.changes[:0]
	b.changedIDs = b.changedIDs[:0]

	for entry := range b.query.Execute() {
		id := entry.ID()
		collider, _ := entry.Component[Collider]()
		if !collider.Active {
			continue
		}
		transform, _ := entry.Component[core.Transform]()
		b.seen[id] = struct{}{}

		current := b.proxies[id]
		geometryChanged := current == nil ||
			current.position != transform.Position ||
			current.rotation != transform.Rotation ||
			current.collider.Shape != collider.Shape ||
			current.collider.Offset != collider.Offset
		if !geometryChanged && current.collider == collider {
			continue
		}

		next := proxy{
			id:       id,
			collider: collider,
			position: transform.Position,
			rotation: transform.Rotation,
		}
		if geometryChanged {
			shape := transformShape(collider.Shape, collider.Offset, transform)
			if shape.shape == nil || !shape.bounds.IsValid() {
				return nil, fmt.Errorf("collision: entity %d has no usable collider shape", id)
			}
			next.shape, next.bounds = shape.shape, shape.bounds
			if current == nil || current.bounds != next.bounds {
				b.changes = append(b.changes, spatial.Change{
					EntityID: id,
					Bounds:   next.bounds,
				})
			}
		} else {
			next.shape, next.bounds = current.shape, current.bounds
		}
		b.updates = append(b.updates, proxyChange{id: id, value: next})
		b.changedIDs = append(b.changedIDs, id)
	}

	for id := range b.proxies {
		if _, ok := b.seen[id]; ok {
			continue
		}
		b.changes = append(b.changes, spatial.Change{EntityID: id, Remove: true})
	}

	if err := b.grid.ApplyBatch(b.changes); err != nil {
		return nil, fmt.Errorf("collision: apply spatial index changes: %w", err)
	}
	for _, update := range b.updates {
		if current := b.proxies[update.id]; current != nil {
			*current = update.value
		} else {
			value := update.value
			b.proxies[update.id] = &value
		}
	}
	for _, change := range b.changes {
		if change.Remove {
			delete(b.proxies, change.EntityID)
		}
	}
	b.tick = tick
	return b.changedIDs, nil
}

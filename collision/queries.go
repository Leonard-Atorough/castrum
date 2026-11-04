package collision

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

const maxRaycastGridCells = 4096

// Queries provides spatial queries over collider data synchronized by the
// collision system. The zero value is not ready for use; register a value
// returned by [NewQueries] with the world.
type Queries struct {
	broadphase *broadphase
	nearby     []core.EntityID
}

// RaycastFilter selects target collider layers and entities to exclude.
type RaycastFilter struct {
	// Mask selects colliders whose Layers intersect this mask. Collider.Mask
	// does not filter raycasts; it controls pairwise collider interactions.
	Mask uint32
	// Exclude lists entity IDs to ignore.
	Exclude []core.EntityID
}

// RaycastHit describes the closest collider hit by a raycast.
type RaycastHit struct {
	// Entity identifies the entity whose collider was hit.
	Entity core.EntityID
	// Collider is the collider data from the synchronized snapshot that produced the hit.
	Collider Collider
	// Point is the hit location in world coordinates.
	Point geom.Vector2
	// Normal is the hit surface normal in world coordinates.
	Normal geom.Vector2
	// Distance is the distance from the segment start to Point.
	Distance float64
}

// NewQueries creates an empty spatial query resource for a world. Register
// the returned resource before running [NewSystem].
func NewQueries() (*Queries, error) {
	bp, err := newBroadphase()
	if err != nil {
		return nil, err
	}
	return &Queries{broadphase: bp}, nil
}

// Tick reports the fixed tick of the latest successful broadphase sync. It is
// zero before the collision system's first successful update.
func (q *Queries) Tick() uint64 {
	if q == nil || q.broadphase == nil {
		return 0
	}
	return q.broadphase.tick
}

// Raycast returns the closest hit along segment among colliders whose Layers
// intersect filter.Mask. It reads the latest successful collision-system sync;
// before the first sync, it returns no hit.
func (q *Queries) Raycast(segment geom.Segment, filter RaycastFilter) (RaycastHit, bool) {
	if q == nil || q.broadphase == nil || q.broadphase.grid == nil ||
		!segment.BoundingBox().IsValid() || !isFinite(segment.Length()) {
		return RaycastHit{}, false
	}
	direction := segment.End.Sub(segment.Start)
	if !isFinite(direction.X) || !isFinite(direction.Y) {
		return RaycastHit{}, false
	}

	bounds := segment.BoundingBox()
	q.nearby = q.candidates(bounds)
	var best RaycastHit
	bestT := math.Inf(1)
	found := false
	for _, id := range q.nearby {
		proxy := q.broadphase.proxies[id]
		if proxy == nil || proxy.collider.Layers&filter.Mask == 0 {
			continue
		}
		excluded := false
		for _, excludedID := range filter.Exclude {
			if id == excludedID {
				excluded = true
				break
			}
		}
		if excluded || !proxy.bounds.OverlapsOrTouches(bounds) {
			continue
		}

		hit := segmentShapeHit(segment, proxy.shape)
		if !hit.hit || (found && (hit.t > bestT || (hit.t == bestT && id >= best.Entity))) {
			continue
		}
		bestT = hit.t
		found = true
		best = RaycastHit{
			Entity:   id,
			Collider: proxy.collider,
			Point:    hit.point,
			Normal:   hit.normal,
			Distance: hit.t * segment.Length(),
		}
	}
	return best, found
}

func (q *Queries) candidates(bounds geom.Rect) []core.EntityID {
	q.nearby = q.nearby[:0]
	minX := math.Floor(bounds.Min.X / cellSize)
	maxX := math.Floor(bounds.Max.X / cellSize)
	minY := math.Floor(bounds.Min.Y / cellSize)
	maxY := math.Floor(bounds.Max.Y / cellSize)
	cellsX, cellsY := maxX-minX+1, maxY-minY+1
	if cellsX > maxRaycastGridCells || cellsY > maxRaycastGridCells ||
		cellsX*cellsY > maxRaycastGridCells {
		for id, proxy := range q.broadphase.proxies {
			if proxy.bounds.OverlapsOrTouches(bounds) {
				q.nearby = append(q.nearby, id)
			}
		}
		return q.nearby
	}
	return q.broadphase.grid.QueryOverlapsInto(bounds, q.nearby)
}

func (q *Queries) sync(world *core.World, tick uint64) ([]core.EntityID, error) {
	if q == nil || q.broadphase == nil {
		return nil, fmt.Errorf("collision: invalid Queries resource; provide a value from NewQueries")
	}
	return q.broadphase.Sync(world, tick)
}

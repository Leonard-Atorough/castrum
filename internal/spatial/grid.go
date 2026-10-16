// Package spatial provides a uniform grid for broad-phase spatial queries.
// [Grid] indexes entity bounds, and callers refine its candidates with exact
// intersection tests.
package spatial

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// GridCell identifies a cell in grid coordinates.
type GridCell struct {
	// X is the cell's horizontal grid coordinate.
	X int
	// Y is the cell's vertical grid coordinate.
	Y int
}

// Grid indexes entity IDs by conservative axis-aligned bounds. It
// returns broad-phase candidates only; callers must perform any
// exact intersection tests themselves.
type Grid struct {
	// cellSize is the width and height of each cell in world units.
	cellSize float64
	// cells maps each occupied cell to its entity IDs.
	cells map[GridCell][]*gridEntry
	// entities maps each indexed entity to its occupied cell range.
	entities map[ecs.EntityID]*gridEntry
}

type gridEntry struct {
	id       ecs.EntityID
	min, max GridCell
	bounds   geom.Rect
}

func (e *gridEntry) contains(cell GridCell) bool {
	return cell.X >= e.min.X && cell.X <= e.max.X && cell.Y >= e.min.Y && cell.Y <= e.max.Y
}

// NewGrid returns a grid whose cells have width and height cellSize in world
// units. cellSize must be positive and finite.
func NewGrid(cellSize float64) (*Grid, error) {
	if cellSize <= 0 || math.IsNaN(cellSize) || math.IsInf(cellSize, 0) {
		return nil, fmt.Errorf("cellSize must be positive, got %f", cellSize)
	}
	return &Grid{
		cellSize: cellSize,
		cells:    make(map[GridCell][]*gridEntry),
		entities: make(map[ecs.EntityID]*gridEntry),
	}, nil
}

// Update replaces entityID's indexed bounds. Invalid bounds return an error
// without changing the current index for that entity.
func (idx *Grid) Update(entityID ecs.EntityID, bounds geom.Rect) error {
	if !bounds.IsValid() {
		return fmt.Errorf("invalid bounds for entity %d: %v", entityID, bounds)
	}

	minCell := idx.worldToGrid(bounds.Min)
	maxCell := idx.worldToGrid(bounds.Max)
	entry, exists := idx.entities[entityID]
	next := gridEntry{id: entityID, min: minCell, max: maxCell, bounds: bounds}
	if exists {
		if entry.min == minCell && entry.max == maxCell {
			entry.bounds = bounds
			return nil
		}
		for x := entry.min.X; x <= entry.max.X; x++ {
			for y := entry.min.Y; y <= entry.max.Y; y++ {
				cell := GridCell{X: x, Y: y}
				if !next.contains(cell) {
					idx.removeFromCell(cell, entityID)
				}
			}
		}
	} else {
		entry = &gridEntry{id: entityID}
		idx.entities[entityID] = entry
	}
	for x := minCell.X; x <= maxCell.X; x++ {
		for y := minCell.Y; y <= maxCell.Y; y++ {
			cell := GridCell{X: x, Y: y}
			if exists && entry.contains(cell) {
				continue
			}
			idx.cells[cell] = append(idx.cells[cell], entry)
		}
	}
	*entry = next
	return nil
}

// Query returns the unique entity IDs indexed in cells overlapped by bounds.
// Results are broad-phase candidates; exact intersection is not tested, and
// result order is unspecified.
func (idx *Grid) Query(bounds geom.Rect) []ecs.EntityID {
	return idx.QueryInto(bounds, nil)
}

// QueryInto returns the unique entity IDs indexed in cells overlapped by
// bounds, reusing ids' backing array when possible. It discards the slice's
// previous contents. Invalid bounds produce an empty result. Results are
// broad-phase candidates, and their order is unspecified.
func (idx *Grid) QueryInto(bounds geom.Rect, ids []ecs.EntityID) []ecs.EntityID {
	return idx.queryInto(bounds, ids, false)
}

// QueryOverlapsInto returns unique entity IDs whose indexed bounds overlap or
// touch bounds, reusing ids' backing array. It discards ids' previous contents.
// Invalid bounds produce an empty result. Order is unspecified; callers must
// still test the exact shapes.
func (idx *Grid) QueryOverlapsInto(bounds geom.Rect, ids []ecs.EntityID) []ecs.EntityID {
	return idx.queryInto(bounds, ids, true)
}

func (idx *Grid) queryInto(bounds geom.Rect, ids []ecs.EntityID, overlapsOnly bool) []ecs.EntityID {
	if !bounds.IsValid() {
		return ids[:0]
	}
	minCell := idx.worldToGrid(bounds.Min)
	maxCell := idx.worldToGrid(bounds.Max)
	ids = ids[:0]

	for x := minCell.X; x <= maxCell.X; x++ {
		for y := minCell.Y; y <= maxCell.Y; y++ {
			cell := GridCell{X: x, Y: y}
			for _, entry := range idx.cells[cell] {
				// Emit at the first shared cell, without a per-query seen map.
				if x == max(minCell.X, entry.min.X) && y == max(minCell.Y, entry.min.Y) {
					// Both bounds are validated before reaching this loop.
					if overlapsOnly && (bounds.Max.X < entry.bounds.Min.X || entry.bounds.Max.X < bounds.Min.X ||
						bounds.Max.Y < entry.bounds.Min.Y || entry.bounds.Max.Y < bounds.Min.Y) {
						continue
					}
					ids = append(ids, entry.id)
				}
			}
		}
	}

	return ids
}

func (idx *Grid) worldToGrid(pos geom.Vector2) GridCell {
	return GridCell{
		X: int(math.Floor(pos.X / idx.cellSize)),
		Y: int(math.Floor(pos.Y / idx.cellSize)),
	}
}

// Remove deletes entityID from the index. It has no effect if the entity is
// not indexed.
func (idx *Grid) Remove(entityID ecs.EntityID) {
	entry := idx.entities[entityID]
	if entry == nil {
		return
	}
	delete(idx.entities, entityID)
	for x := entry.min.X; x <= entry.max.X; x++ {
		for y := entry.min.Y; y <= entry.max.Y; y++ {
			idx.removeFromCell(GridCell{X: x, Y: y}, entityID)
		}
	}
}

func (idx *Grid) removeFromCell(cell GridCell, id ecs.EntityID) {
	entries := idx.cells[cell]
	for i, entry := range entries {
		if entry.id == id {
			last := len(entries) - 1
			entries[i] = entries[last]
			entries[last] = nil
			if last == 0 {
				delete(idx.cells, cell)
			} else {
				idx.cells[cell] = entries[:last]
			}
			return
		}
	}
}

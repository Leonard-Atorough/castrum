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

const maxCellsPerRange = 4096

// GridCell identifies a cell in grid coordinates.
type GridCell struct {
	// X is the cell's horizontal grid coordinate.
	X int
	// Y is the cell's vertical grid coordinate.
	Y int
}

// Grid indexes entity IDs by conservative axis-aligned bounds. It
// returns broad-phase candidates only; callers must perform any
// exact intersection tests themselves. Large bounds and query ranges use
// entry scans rather than expanding every covered cell.
type Grid struct {
	// cellSize is the width and height of each cell in world units.
	cellSize float64
	// cells maps occupied cells to pointers to entries owned by entities.
	cells map[GridCell][]*gridEntry
	// entities owns the indexed entries, keyed by entity ID.
	entities map[ecs.EntityID]*gridEntry
	// oversized holds entries whose ranges exceed the cell coverage budget.
	oversized map[ecs.EntityID]*gridEntry
	// batchEntries and batchIDs are reused by ApplyBatch.
	batchEntries []gridEntry
	batchIDs     map[ecs.EntityID]struct{}
}

type gridEntry struct {
	id        ecs.EntityID
	min, max  GridCell
	bounds    geom.Rect
	oversized bool
}

// Change updates or removes one entity's indexed bounds.
type Change struct {
	EntityID ecs.EntityID
	Bounds   geom.Rect
	Remove   bool
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
		cellSize:     cellSize,
		cells:        make(map[GridCell][]*gridEntry),
		entities:     make(map[ecs.EntityID]*gridEntry),
		oversized:    make(map[ecs.EntityID]*gridEntry),
		batchEntries: make([]gridEntry, 0, 16),
		batchIDs:     make(map[ecs.EntityID]struct{}),
	}, nil
}

// Update replaces entityID's indexed bounds. Invalid bounds return an error
// without changing the current index for that entity.
func (idx *Grid) Update(entityID ecs.EntityID, bounds geom.Rect) error {
	entry, err := idx.prepareEntry(entityID, bounds)
	if err != nil {
		return err
	}
	idx.updateEntry(entry)
	return nil
}

// ApplyBatch applies all changes after validating the complete batch. If
// validation fails, the grid is unchanged. Entity IDs must be unique in the
// batch.
func (idx *Grid) ApplyBatch(changes []Change) error {
	if len(changes) == 0 {
		return nil
	}
	if len(changes) > cap(idx.batchEntries) {
		idx.batchEntries = make([]gridEntry, len(changes))
	} else {
		idx.batchEntries = idx.batchEntries[:len(changes)]
	}
	entries := idx.batchEntries
	clear(idx.batchIDs)
	for i, change := range changes {
		if _, ok := idx.batchIDs[change.EntityID]; ok {
			return fmt.Errorf("duplicate entity %d in grid change batch", change.EntityID)
		}
		idx.batchIDs[change.EntityID] = struct{}{}
		if change.Remove {
			continue
		}
		entry, err := idx.prepareEntry(change.EntityID, change.Bounds)
		if err != nil {
			return err
		}
		entries[i] = entry
	}
	idx.batchEntries = entries
	for i, change := range changes {
		if change.Remove {
			idx.Remove(change.EntityID)
			continue
		}
		idx.updateEntry(entries[i])
	}
	return nil
}

func (idx *Grid) prepareEntry(entityID ecs.EntityID, bounds geom.Rect) (gridEntry, error) {
	if !bounds.IsValid() {
		return gridEntry{}, fmt.Errorf("invalid bounds for entity %d: %v", entityID, bounds)
	}
	minCell, err := idx.worldToGrid(bounds.Min)
	if err != nil {
		return gridEntry{}, fmt.Errorf("invalid bounds for entity %d: %w", entityID, err)
	}
	maxCell, err := idx.worldToGrid(bounds.Max)
	if err != nil {
		return gridEntry{}, fmt.Errorf("invalid bounds for entity %d: %w", entityID, err)
	}
	return gridEntry{
		id: entityID, min: minCell, max: maxCell, bounds: bounds,
		oversized: exceedsCellBudget(minCell, maxCell),
	}, nil
}

func exceedsCellBudget(minCell, maxCell GridCell) bool {
	// Unsigned differences also handle ranges spanning the signed integer limits.
	width := uint(maxCell.X) - uint(minCell.X)
	height := uint(maxCell.Y) - uint(minCell.Y)
	return width >= maxCellsPerRange || height >= maxCellsPerRange ||
		(width+1)*(height+1) > maxCellsPerRange
}

func (idx *Grid) updateEntry(next gridEntry) {
	entityID := next.id
	minCell, maxCell := next.min, next.max
	entry, exists := idx.entities[entityID]
	if exists {
		if entry.min == minCell && entry.max == maxCell {
			entry.bounds = next.bounds
			return
		}
		if entry.oversized {
			delete(idx.oversized, entityID)
		} else {
			for x := entry.min.X; x <= entry.max.X; x++ {
				for y := entry.min.Y; y <= entry.max.Y; y++ {
					cell := GridCell{X: x, Y: y}
					if next.oversized || !next.contains(cell) {
						idx.removeFromCell(cell, entityID)
					}
				}
			}
		}
	} else {
		entry = &gridEntry{id: entityID}
		idx.entities[entityID] = entry
	}
	if next.oversized {
		idx.oversized[entityID] = entry
	} else {
		for x := minCell.X; x <= maxCell.X; x++ {
			for y := minCell.Y; y <= maxCell.Y; y++ {
				cell := GridCell{X: x, Y: y}
				if exists && !entry.oversized && entry.contains(cell) {
					continue
				}
				idx.cells[cell] = append(idx.cells[cell], entry)
			}
		}
	}
	*entry = next
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
	minCell, err := idx.worldToGrid(bounds.Min)
	if err != nil {
		return ids[:0]
	}
	maxCell, err := idx.worldToGrid(bounds.Max)
	if err != nil {
		return ids[:0]
	}
	ids = ids[:0]

	entries := idx.oversized
	if exceedsCellBudget(minCell, maxCell) {
		entries = idx.entities
	} else {
		for x := minCell.X; x <= maxCell.X; x++ {
			for y := minCell.Y; y <= maxCell.Y; y++ {
				cell := GridCell{X: x, Y: y}
				for _, entry := range idx.cells[cell] {
					// Emit at the first shared cell, without a per-query seen map.
					if x == max(minCell.X, entry.min.X) && y == max(minCell.Y, entry.min.Y) {
						if overlapsOnly && !bounds.OverlapsOrTouches(entry.bounds) {
							continue
						}
						ids = append(ids, entry.id)
					}
				}
			}
		}
	}
	for _, entry := range entries {
		if entry.max.X < minCell.X || entry.min.X > maxCell.X ||
			entry.max.Y < minCell.Y || entry.min.Y > maxCell.Y {
			continue
		}
		if overlapsOnly && !bounds.OverlapsOrTouches(entry.bounds) {
			continue
		}
		ids = append(ids, entry.id)
	}

	return ids
}

func (idx *Grid) worldToGrid(pos geom.Vector2) (GridCell, error) {
	x, y := math.Floor(pos.X/idx.cellSize), math.Floor(pos.Y/idx.cellSize)
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	if x < float64(minInt) || x >= float64(maxInt) ||
		y < float64(minInt) || y >= float64(maxInt) {
		return GridCell{}, fmt.Errorf("grid coordinates exceed integer range")
	}
	return GridCell{X: int(x), Y: int(y)}, nil
}

// Remove deletes entityID from the index. It has no effect if the entity is
// not indexed.
func (idx *Grid) Remove(entityID ecs.EntityID) {
	entry := idx.entities[entityID]
	if entry == nil {
		return
	}
	delete(idx.entities, entityID)
	if entry.oversized {
		delete(idx.oversized, entityID)
		return
	}
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

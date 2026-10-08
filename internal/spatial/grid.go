// Package spatial provides generic bounds-based spatial indexes.
package spatial

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// GridCell identifies one cell in world-space grid coordinates.
type GridCell struct {
	X, Y int
}

// Grid indexes entity IDs by conservative axis-aligned bounds. It
// returns broad-phase candidates only; callers must perform any
// exact intersection tests themselves.
type Grid struct {
	cellSize float64
	cells    map[GridCell]map[ecs.EntityID]struct{}
	entities map[ecs.EntityID]map[GridCell]struct{}
	seen     map[ecs.EntityID]struct{}
}

// NewGrid creates a bounds-based grid with cells of cellSize world units.
func NewGrid(cellSize float64) (*Grid, error) {
	if cellSize <= 0 || math.IsNaN(cellSize) || math.IsInf(cellSize, 0) {
		return nil, fmt.Errorf("cellSize must be positive, got %f", cellSize)
	}
	return &Grid{
		cellSize: cellSize,
		cells:    make(map[GridCell]map[ecs.EntityID]struct{}),
		entities: make(map[ecs.EntityID]map[GridCell]struct{}),
		seen:     make(map[ecs.EntityID]struct{}),
	}, nil
}

// Update replaces entityID's indexed bounds and removes its old cell
// memberships first.
func (idx *Grid) Update(entityID ecs.EntityID, bounds geom.Rect) error {
	if !bounds.IsValid() {
		return fmt.Errorf("invalid bounds for entity %d: %v", entityID, bounds)
	}

	// Reuse the entity's membership set while replacing its cell memberships.
	cells := idx.detach(entityID)
	if cells == nil {
		cells = make(map[GridCell]struct{})
	}
	minCell := idx.worldToGrid(bounds.Min)
	maxCell := idx.worldToGrid(bounds.Max)
	for x := minCell.X; x <= maxCell.X; x++ {
		for y := minCell.Y; y <= maxCell.Y; y++ {
			cell := GridCell{X: x, Y: y}
			if idx.cells[cell] == nil {
				idx.cells[cell] = make(map[ecs.EntityID]struct{})
			}
			idx.cells[cell][entityID] = struct{}{}
			cells[cell] = struct{}{}
		}
	}
	idx.entities[entityID] = cells
	return nil
}

// Query returns all entity IDs in cells overlapped by bounds.
func (idx *Grid) Query(bounds geom.Rect) []ecs.EntityID {
	return idx.QueryInto(bounds, nil)
}

// QueryInto fills ids with the candidates overlapping bounds,
// reusing ids' backing array when possible and discarding whatever
// it held. Results are deduplicated when an entity spans cells.
func (idx *Grid) QueryInto(bounds geom.Rect, ids []ecs.EntityID) []ecs.EntityID {
	if !bounds.IsValid() {
		return ids[:0]
	}
	minCell := idx.worldToGrid(bounds.Min)
	maxCell := idx.worldToGrid(bounds.Max)
	// A multi-cell entity can appear repeatedly while scanning the cell
	// range. Keep a per-query set so each candidate is returned once.
	clear(idx.seen)
	ids = ids[:0]

	for x := minCell.X; x <= maxCell.X; x++ {
		for y := minCell.Y; y <= maxCell.Y; y++ {
			cell := GridCell{X: x, Y: y}
			if entities, exists := idx.cells[cell]; exists {
				for entityID := range entities {
					if _, exists := idx.seen[entityID]; exists {
						continue
					}
					idx.seen[entityID] = struct{}{}
					ids = append(ids, entityID)
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

// Remove deletes entityID from every cell in which it is indexed.
func (idx *Grid) Remove(entityID ecs.EntityID) {
	idx.detach(entityID)
}

func (idx *Grid) detach(entityID ecs.EntityID) map[GridCell]struct{} {
	cells, exists := idx.entities[entityID]
	if !exists {
		return nil
	}
	delete(idx.entities, entityID)
	for cell := range cells {
		delete(idx.cells[cell], entityID)
		if len(idx.cells[cell]) == 0 {
			delete(idx.cells, cell)
		}
	}
	clear(cells)
	return cells
}

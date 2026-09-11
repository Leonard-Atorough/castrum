package physics

import (
	"fmt"
	"math"

	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

type GridCell struct {
	X, Y int
}

type SpatialIndex struct {
	cellSize float64
	cells    map[GridCell]map[ecs.EntityID]bool
	entities map[ecs.EntityID]GridCell
	seen     map[ecs.EntityID]struct{}
}

func newIndex(cellSize float64) (*SpatialIndex, error) {
	if cellSize <= 0 {
		return nil, fmt.Errorf("cellSize must be positive, got %f", cellSize)
	}
	return &SpatialIndex{
		cellSize: cellSize,
		cells:    make(map[GridCell]map[ecs.EntityID]bool),
		entities: make(map[ecs.EntityID]GridCell),
		seen:     make(map[ecs.EntityID]struct{}),
	}, nil
}

func (idx *SpatialIndex) Update(entityID ecs.EntityID, pos geom.Vector2) error {
	if math.IsNaN(pos.X) || math.IsNaN(pos.Y) || math.IsInf(pos.X, 0) || math.IsInf(pos.Y, 0) {
		return fmt.Errorf("invalid position for entity %d: %v", entityID, pos)
	}
	newCell := idx.worldToGrid(pos)
	oldCell, exists := idx.entities[entityID]

	if exists {
		if oldCell == newCell {
			return nil
		}

		delete(idx.cells[oldCell], entityID)
		if len(idx.cells[oldCell]) == 0 {
			delete(idx.cells, oldCell)
		}
	}

	if idx.cells[newCell] == nil {
		idx.cells[newCell] = make(map[ecs.EntityID]bool)
	}
	idx.cells[newCell][entityID] = true
	idx.entities[entityID] = newCell
	return nil
}

func (idx *SpatialIndex) Query(pos geom.Vector2, radius float64) []ecs.EntityID {
	return idx.QueryInto(pos, radius, nil)
}

func (idx *SpatialIndex) QueryInto(pos geom.Vector2, radius float64, ids []ecs.EntityID) []ecs.EntityID {
	centerCell := idx.worldToGrid(pos)
	radiusInCells := int(math.Ceil(radius / idx.cellSize))
	clear(idx.seen)
	ids = ids[:0]

	for x := centerCell.X - radiusInCells; x <= centerCell.X+radiusInCells; x++ {
		for y := centerCell.Y - radiusInCells; y <= centerCell.Y+radiusInCells; y++ {
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

func (idx *SpatialIndex) worldToGrid(pos geom.Vector2) GridCell {
	return GridCell{
		X: int(math.Floor(pos.X / idx.cellSize)),
		Y: int(math.Floor(pos.Y / idx.cellSize)),
	}
}

func (idx *SpatialIndex) Remove(entityID ecs.EntityID) {
	cell, exists := idx.entities[entityID]
	if !exists {
		return
	}
	delete(idx.cells[cell], entityID)
	if len(idx.cells[cell]) == 0 {
		delete(idx.cells, cell)
	}
	delete(idx.entities, entityID)
}

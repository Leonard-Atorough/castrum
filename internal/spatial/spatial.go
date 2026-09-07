package spatial

import (
	"fmt"
	"math"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

type GridCell struct {
	X, Y int
}

type SpatialIndex struct {
	cellSize float64
	cells    map[GridCell]map[core.EntityID]bool
	entities map[core.EntityID]GridCell
}

func NewIndex(cellSize float64) (*SpatialIndex, error) {
	if cellSize <= 0 {
		return nil, fmt.Errorf("cellSize must be positive, got %f", cellSize)
	}
	return &SpatialIndex{
		cellSize: cellSize,
		cells:    make(map[GridCell]map[core.EntityID]bool),
		entities: make(map[core.EntityID]GridCell),
	}, nil
}

func (idx *SpatialIndex) Update(entityID core.EntityID, pos geom.Vector2) error {
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
		idx.cells[newCell] = make(map[core.EntityID]bool)
	}
	idx.cells[newCell][entityID] = true
	idx.entities[entityID] = newCell
	return nil
}

func (idx *SpatialIndex) Query(pos geom.Vector2, radius float64) []core.EntityID {
	centerCell := idx.worldToGrid(pos)
	radiusInCells := int(math.Ceil(radius / idx.cellSize))
	results := make(map[core.EntityID]bool)

	for x := centerCell.X - radiusInCells; x <= centerCell.X+radiusInCells; x++ {
		for y := centerCell.Y - radiusInCells; y <= centerCell.Y+radiusInCells; y++ {
			cell := GridCell{X: x, Y: y}
			if entities, exists := idx.cells[cell]; exists {
				for entityID := range entities {
					results[entityID] = true
				}
			}
		}
	}

	ids := make([]core.EntityID, 0, len(results))
	for id := range results {
		ids = append(ids, id)
	}
	return ids
}

func (idx *SpatialIndex) worldToGrid(pos geom.Vector2) GridCell {
	return GridCell{
		X: int(math.Floor(pos.X / idx.cellSize)),
		Y: int(math.Floor(pos.Y / idx.cellSize)),
	}
}

func (idx *SpatialIndex) Remove(entityID core.EntityID) {
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

type SpatialIndexHandler struct {
	Index *SpatialIndex
}

func NewManager(cellSize float64) (*SpatialIndexHandler, error) {
	idx, err := NewIndex(cellSize)
	if err != nil {
		return nil, err
	}
	return &SpatialIndexHandler{
		Index: idx,
	}, nil
}

func (mgr *SpatialIndexHandler) Update(world *core.World, deltaTime float64) error {
	transforms := core.QueryFor[components.Transform](world)

	for _, entityID := range transforms {
		transform, _ := world.GetComponent[components.Transform](entityID)
		if err := mgr.Index.Update(entityID, transform.Position); err != nil {
			return err
		}
	}
	return nil
}

func (mgr *SpatialIndexHandler) RemoveEntity(entityID core.EntityID) {
	mgr.Index.Remove(entityID)
}

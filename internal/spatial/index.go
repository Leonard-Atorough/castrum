package spatial

import (
	"math"

	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

type GridCell struct {
	X, Y int
}

type Index struct {
	cellSize float64
	cells    map[GridCell]map[core.EntityID]bool
	entities map[core.EntityID]GridCell
}

func NewIndex(cellSize float64) *Index {
	return &Index{
		cellSize: cellSize,
		cells:    make(map[GridCell]map[core.EntityID]bool),
		entities: make(map[core.EntityID]GridCell),
	}
}

func (idx *Index) Update(entityID core.EntityID, pos geom.Vector2) {
	newCell := idx.worldToGrid(pos)
	oldCell, exists := idx.entities[entityID]

	if exists {
		if oldCell == newCell {
			return
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
}

func (idx *Index) Query(pos geom.Vector2, radius float64) []core.EntityID {
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

func (idx *Index) worldToGrid(pos geom.Vector2) GridCell {
	return GridCell{
		X: int(pos.X / idx.cellSize),
		Y: int(pos.Y / idx.cellSize),
	}
}

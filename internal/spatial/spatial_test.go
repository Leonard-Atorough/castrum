package spatial

import (
	"math"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

func TestNewIndex(t *testing.T) {
	tests := []struct {
		name     string
		cellSize float64
		wantErr  bool
	}{
		{
			name:     "valid cell size",
			cellSize: 10.0,
			wantErr:  false,
		},
		{
			name:     "small cell size",
			cellSize: 0.1,
			wantErr:  false,
		},
		{
			name:     "zero cell size",
			cellSize: 0.0,
			wantErr:  true,
		},
		{
			name:     "negative cell size",
			cellSize: -5.0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := NewIndex(tt.cellSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && idx == nil {
				t.Error("NewIndex() returned nil index when expected success")
			}
		})
	}
}

func TestSpatialIndex_Update(t *testing.T) {
	idx, _ := NewIndex(10.0)

	tests := []struct {
		name      string
		entityID  ecs.EntityID
		pos       geom.Vector2
		wantErr   bool
		expectKey bool
	}{
		{
			name:      "new entity at origin",
			entityID:  1,
			pos:       geom.Vector2{X: 0, Y: 0},
			wantErr:   false,
			expectKey: true,
		},
		{
			name:      "new entity positive coordinates",
			entityID:  2,
			pos:       geom.Vector2{X: 25.0, Y: 35.0},
			wantErr:   false,
			expectKey: true,
		},
		{
			name:      "new entity negative coordinates",
			entityID:  3,
			pos:       geom.Vector2{X: -15.0, Y: -25.0},
			wantErr:   false,
			expectKey: true,
		},
		{
			name:      "NaN X coordinate",
			entityID:  4,
			pos:       geom.Vector2{X: math.NaN(), Y: 0},
			wantErr:   true,
			expectKey: false,
		},
		{
			name:      "NaN Y coordinate",
			entityID:  5,
			pos:       geom.Vector2{X: 0, Y: math.NaN()},
			wantErr:   true,
			expectKey: false,
		},
		{
			name:      "Inf X coordinate positive",
			entityID:  6,
			pos:       geom.Vector2{X: math.Inf(1), Y: 0},
			wantErr:   true,
			expectKey: false,
		},
		{
			name:      "Inf X coordinate negative",
			entityID:  7,
			pos:       geom.Vector2{X: math.Inf(-1), Y: 0},
			wantErr:   true,
			expectKey: false,
		},
		{
			name:      "Inf Y coordinate positive",
			entityID:  8,
			pos:       geom.Vector2{X: 0, Y: math.Inf(1)},
			wantErr:   true,
			expectKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := idx.Update(tt.entityID, tt.pos)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			_, exists := idx.entities[tt.entityID]
			if exists != tt.expectKey {
				t.Errorf("Update() entity in map = %v, want %v", exists, tt.expectKey)
			}
		})
	}
}

func TestSpatialIndex_UpdateExistingEntity(t *testing.T) {
	idx, _ := NewIndex(10.0)
	entityID := ecs.EntityID(1)

	// Add entity
	pos1 := geom.Vector2{X: 5.0, Y: 5.0}
	err := idx.Update(entityID, pos1)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	cell1 := idx.worldToGrid(pos1)
	if _, exists := idx.cells[cell1][entityID]; !exists {
		t.Error("Update() entity not in expected cell")
	}

	// Move entity to different cell
	pos2 := geom.Vector2{X: 25.0, Y: 25.0}
	err = idx.Update(entityID, pos2)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	cell2 := idx.worldToGrid(pos2)
	if _, exists := idx.cells[cell2][entityID]; !exists {
		t.Error("Update() entity not in new cell")
	}

	if _, exists := idx.cells[cell1][entityID]; exists {
		t.Error("Update() entity still in old cell")
	}

	// Move entity to same cell
	pos3 := geom.Vector2{X: 26.0, Y: 26.0}
	err = idx.Update(entityID, pos3)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify entity still in same cell
	if _, exists := idx.cells[cell2][entityID]; !exists {
		t.Error("Update() entity not in expected cell after movement within cell")
	}
}

func TestSpatialIndex_Query(t *testing.T) {
	idx, _ := NewIndex(10.0)

	// Add some entities
	entities := map[ecs.EntityID]geom.Vector2{
		1: {X: 0, Y: 0},
		2: {X: 5, Y: 5},
		3: {X: 15, Y: 0},
		4: {X: -15, Y: 0},
		5: {X: 100, Y: 100},
	}

	for id, pos := range entities {
		if err := idx.Update(id, pos); err != nil {
			t.Fatalf("Update() failed: %v", err)
		}
	}

	tests := []struct {
		name          string
		queryPos      geom.Vector2
		radius        float64
		expectedCount int
		shouldContain []ecs.EntityID
	}{
		{
			name:          "query at origin with small radius",
			queryPos:      geom.Vector2{X: 0, Y: 0},
			radius:        5.0,
			expectedCount: 3,
			shouldContain: []ecs.EntityID{1, 2, 3},
		},
		{
			name:          "query at origin with large radius",
			queryPos:      geom.Vector2{X: 0, Y: 0},
			radius:        20.0,
			expectedCount: 4,
			shouldContain: []ecs.EntityID{1, 2, 3, 4},
		},
		{
			name:          "query with zero radius",
			queryPos:      geom.Vector2{X: 0, Y: 0},
			radius:        0.0,
			expectedCount: 2,
			shouldContain: []ecs.EntityID{1, 2},
		},
		{
			name:          "query far away",
			queryPos:      geom.Vector2{X: 200, Y: 200},
			radius:        50.0,
			expectedCount: 0,
			shouldContain: []ecs.EntityID{},
		},
		{
			name:          "query at entity location",
			queryPos:      geom.Vector2{X: 15, Y: 0},
			radius:        5.0,
			expectedCount: 3,
			shouldContain: []ecs.EntityID{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := idx.Query(tt.queryPos, tt.radius)

			if len(results) != tt.expectedCount {
				t.Errorf("Query() returned %d entities, want %d", len(results), tt.expectedCount)
			}

			resultMap := make(map[ecs.EntityID]bool)
			for _, id := range results {
				resultMap[id] = true
			}

			for _, id := range tt.shouldContain {
				if !resultMap[id] {
					t.Errorf("Query() missing entity %d", id)
				}
			}
		})
	}
}

func TestSpatialIndex_QueryEmpty(t *testing.T) {
	idx, _ := NewIndex(10.0)

	results := idx.Query(geom.Vector2{X: 0, Y: 0}, 10.0)
	if len(results) != 0 {
		t.Errorf("Query() on empty index returned %d entities, want 0", len(results))
	}
}

func TestSpatialIndex_Remove(t *testing.T) {
	idx, _ := NewIndex(10.0)
	entityID := ecs.EntityID(1)

	// Remove from empty index
	idx.Remove(entityID)
	if _, exists := idx.entities[entityID]; exists {
		t.Error("Remove() added entity that should not exist")
	}

	// Add entity and remove it
	pos := geom.Vector2{X: 5.0, Y: 5.0}
	idx.Update(entityID, pos)

	idx.Remove(entityID)
	if _, exists := idx.entities[entityID]; exists {
		t.Error("Remove() did not remove entity from entities map")
	}

	cell := idx.worldToGrid(pos)
	if _, exists := idx.cells[cell][entityID]; exists {
		t.Error("Remove() did not remove entity from cell")
	}
}

func TestSpatialIndex_RemoveCleanupEmptyCells(t *testing.T) {
	idx, _ := NewIndex(10.0)
	entityID := ecs.EntityID(1)

	pos := geom.Vector2{X: 5.0, Y: 5.0}
	idx.Update(entityID, pos)

	cell := idx.worldToGrid(pos)
	if _, exists := idx.cells[cell]; !exists {
		t.Error("Update() did not create cell")
	}

	idx.Remove(entityID)

	if _, exists := idx.cells[cell]; exists {
		t.Error("Remove() did not clean up empty cell")
	}
}

func TestSpatialIndex_RemoveMultipleEntitiesInCell(t *testing.T) {
	idx, _ := NewIndex(10.0)

	// Add multiple entities in same cell
	id1 := ecs.EntityID(1)
	id2 := ecs.EntityID(2)
	pos1 := geom.Vector2{X: 5.0, Y: 5.0}
	pos2 := geom.Vector2{X: 7.0, Y: 7.0}

	idx.Update(id1, pos1)
	idx.Update(id2, pos2)

	cell := idx.worldToGrid(pos1)

	// Remove first entity
	idx.Remove(id1)

	// Cell should still exist with second entity
	if _, exists := idx.cells[cell][id2]; !exists {
		t.Error("Remove() removed cell that still has entities")
	}

	// Remove second entity
	idx.Remove(id2)

	// Now cell should be cleaned up
	if _, exists := idx.cells[cell]; exists {
		t.Error("Remove() did not clean up cell after removing all entities")
	}
}

func TestSpatialIndex_WorldToGrid(t *testing.T) {
	idx, _ := NewIndex(10.0)

	tests := []struct {
		name     string
		pos      geom.Vector2
		expected GridCell
	}{
		{
			name:     "origin",
			pos:      geom.Vector2{X: 0, Y: 0},
			expected: GridCell{X: 0, Y: 0},
		},
		{
			name:     "positive coordinates within cell",
			pos:      geom.Vector2{X: 5.0, Y: 7.0},
			expected: GridCell{X: 0, Y: 0},
		},
		{
			name:     "positive coordinates next cell",
			pos:      geom.Vector2{X: 10.0, Y: 10.0},
			expected: GridCell{X: 1, Y: 1},
		},
		{
			name:     "positive coordinates second cell",
			pos:      geom.Vector2{X: 25.0, Y: 35.0},
			expected: GridCell{X: 2, Y: 3},
		},
		{
			name:     "negative coordinates",
			pos:      geom.Vector2{X: -5.0, Y: -7.0},
			expected: GridCell{X: -1, Y: -1},
		},
		{
			name:     "negative coordinates boundary",
			pos:      geom.Vector2{X: -10.0, Y: -10.0},
			expected: GridCell{X: -1, Y: -1},
		},
		{
			name:     "mixed sign coordinates",
			pos:      geom.Vector2{X: 15.0, Y: -5.0},
			expected: GridCell{X: 1, Y: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.worldToGrid(tt.pos)
			if result != tt.expected {
				t.Errorf("worldToGrid(%v) = %v, want %v", tt.pos, result, tt.expected)
			}
		})
	}
}

func TestSpatialIndex_DifferentCellSizes(t *testing.T) {
	tests := []struct {
		name     string
		cellSize float64
		pos      geom.Vector2
		expected GridCell
	}{
		{
			name:     "cell size 5",
			cellSize: 5.0,
			pos:      geom.Vector2{X: 12.0, Y: 12.0},
			expected: GridCell{X: 2, Y: 2},
		},
		{
			name:     "cell size 20",
			cellSize: 20.0,
			pos:      geom.Vector2{X: 50.0, Y: 50.0},
			expected: GridCell{X: 2, Y: 2},
		},
		{
			name:     "cell size 0.5",
			cellSize: 0.5,
			pos:      geom.Vector2{X: 2.5, Y: 2.5},
			expected: GridCell{X: 5, Y: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, _ := NewIndex(tt.cellSize)
			result := idx.worldToGrid(tt.pos)
			if result != tt.expected {
				t.Errorf("worldToGrid(%v) with cellSize %f = %v, want %v", tt.pos, tt.cellSize, result, tt.expected)
			}
		})
	}
}

func TestSpatialIndex_QueryRadiusAcrossCells(t *testing.T) {
	idx, _ := NewIndex(10.0)

	// Place entities at cell boundaries
	idx.Update(1, geom.Vector2{X: 9.9, Y: 0})  // Cell (0, 0)
	idx.Update(2, geom.Vector2{X: 10.1, Y: 0}) // Cell (1, 0)
	idx.Update(3, geom.Vector2{X: 20.0, Y: 0}) // Cell (2, 0)

	// Query from position that should reach both sides
	results := idx.Query(geom.Vector2{X: 10.0, Y: 0}, 5.0)

	if len(results) < 2 {
		t.Errorf("Query() across cell boundary returned %d entities, want at least 2", len(results))
	}
}

func TestSpatialIndex_LargeDataset(t *testing.T) {
	idx, _ := NewIndex(50.0)

	// Add 1000 entities in a grid
	for i := 0; i < 1000; i++ {
		x := float64((i % 100) * 5)
		y := float64((i / 100) * 5)
		if err := idx.Update(ecs.EntityID(i), geom.Vector2{X: x, Y: y}); err != nil {
			t.Fatalf("Update() failed: %v", err)
		}
	}

	// Query should be efficient
	results := idx.Query(geom.Vector2{X: 50, Y: 50}, 50.0)

	if len(results) == 0 {
		t.Error("Query() on large dataset returned no results")
	}

	// Verify all entities are findable
	if len(idx.entities) != 1000 {
		t.Errorf("Index has %d entities, want 1000", len(idx.entities))
	}
}

func TestSpatialIndex_UpdateToSamePosition(t *testing.T) {
	idx, _ := NewIndex(10.0)
	entityID := ecs.EntityID(1)
	pos := geom.Vector2{X: 5.0, Y: 5.0}

	// First update
	idx.Update(entityID, pos)
	cell := idx.worldToGrid(pos)

	// Same position update should not error and keep entity in same place
	err := idx.Update(entityID, pos)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	if _, exists := idx.cells[cell][entityID]; !exists {
		t.Error("Update() to same position lost entity")
	}
}

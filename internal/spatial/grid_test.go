package spatial

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

func TestNewGrid(t *testing.T) {
	for _, test := range []struct {
		name     string
		cellSize float64
		wantErr  bool
	}{
		{name: "valid", cellSize: 10},
		{name: "small", cellSize: 0.1},
		{name: "zero", cellSize: 0, wantErr: true},
		{name: "negative", cellSize: -5, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			idx, err := NewGrid(test.cellSize)
			if (err != nil) != test.wantErr {
				t.Fatalf("NewGrid() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && idx == nil {
				t.Fatal("NewGrid() returned nil index")
			}
		})
	}
}

func TestGridUpdateIndexesEveryOverlappedCell(t *testing.T) {
	idx, _ := NewGrid(10)
	bounds := geom.Rect{Min: geom.Vector2{X: -5, Y: 5}, Max: geom.Vector2{X: 25, Y: 15}}

	if err := idx.Update(1, bounds); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	for x := -1; x <= 2; x++ {
		for y := 0; y <= 1; y++ {
			if !slices.Contains(idx.cells[GridCell{X: x, Y: y}], idx.entities[1]) {
				t.Errorf("entity missing from cell (%d, %d)", x, y)
			}
		}
	}
	entry := idx.entities[1]
	if got := (entry.max.X - entry.min.X + 1) * (entry.max.Y - entry.min.Y + 1); got != 8 {
		t.Errorf("entity occupies %d cells, want 8", got)
	}
}

func TestGridUpdateRemovesStaleCells(t *testing.T) {
	idx, _ := NewGrid(10)
	first := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 19, Y: 19}}
	second := geom.Rect{Min: geom.Vector2{X: 30, Y: 30}, Max: geom.Vector2{X: 39, Y: 39}}

	if err := idx.Update(1, first); err != nil {
		t.Fatal(err)
	}
	if err := idx.Update(1, second); err != nil {
		t.Fatal(err)
	}

	if _, ok := idx.cells[GridCell{X: 0, Y: 0}]; ok {
		t.Error("stale cell (0, 0) remains after update")
	}
	if !slices.Contains(idx.cells[GridCell{X: 3, Y: 3}], idx.entities[1]) {
		t.Error("entity missing from updated cell (3, 3)")
	}
}

func TestGridRejectsInvalidBounds(t *testing.T) {
	idx, _ := NewGrid(10)
	invalid := []geom.Rect{
		{Min: geom.Vector2{X: math.NaN()}, Max: geom.Vector2{X: 1, Y: 1}},
		{Min: geom.Vector2{}, Max: geom.Vector2{X: math.Inf(1), Y: 1}},
		{Min: geom.Vector2{X: 2, Y: 2}, Max: geom.Vector2{X: 1, Y: 1}},
	}
	for _, bounds := range invalid {
		if err := idx.Update(1, bounds); err == nil {
			t.Errorf("Update(%v) returned nil error", bounds)
		}
	}
}

func TestGridQueryBoundsDeduplicatesMultiCellEntities(t *testing.T) {
	idx, _ := NewGrid(10)
	if err := idx.Update(1, geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 25, Y: 25}}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Update(2, geom.Rect{Min: geom.Vector2{X: 40, Y: 40}, Max: geom.Vector2{X: 45, Y: 45}}); err != nil {
		t.Fatal(err)
	}

	query := geom.Rect{Min: geom.Vector2{X: 5, Y: 5}, Max: geom.Vector2{X: 35, Y: 35}}
	results := idx.Query(query)
	if len(results) != 1 || results[0] != 1 {
		t.Errorf("Query() = %v, want [1]", results)
	}

	buffer := make([]ecs.EntityID, 0, 4)
	results = idx.QueryInto(query, buffer)
	if len(results) != 1 || results[0] != 1 {
		t.Errorf("QueryInto() = %v, want [1]", results)
	}
}

func TestGridRemoveClearsEveryMembership(t *testing.T) {
	idx, _ := NewGrid(10)
	bounds := geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 25, Y: 25}}
	if err := idx.Update(1, bounds); err != nil {
		t.Fatal(err)
	}

	idx.Remove(1)
	if _, ok := idx.entities[1]; ok {
		t.Error("entity membership remains after Remove")
	}
	if len(idx.cells) != 0 {
		t.Errorf("cells = %d after Remove, want 0", len(idx.cells))
	}
}

func TestGridWorldToGridHandlesNegativeCoordinates(t *testing.T) {
	idx, _ := NewGrid(10)
	for _, test := range []struct {
		pos  geom.Vector2
		want GridCell
	}{
		{pos: geom.Vector2{}, want: GridCell{}},
		{pos: geom.Vector2{X: -0.1, Y: -10}, want: GridCell{X: -1, Y: -1}},
		{pos: geom.Vector2{X: 25, Y: -5}, want: GridCell{X: 2, Y: -1}},
	} {
		if got := idx.worldToGrid(test.pos); got != test.want {
			t.Errorf("worldToGrid(%v) = %v, want %v", test.pos, got, test.want)
		}
	}
}

func TestGridLargeDataset(t *testing.T) {
	idx, _ := NewGrid(50)
	for i := 0; i < 1000; i++ {
		x := float64((i % 100) * 5)
		y := float64((i / 100) * 5)
		bounds := geom.Rect{Min: geom.Vector2{X: x, Y: y}, Max: geom.Vector2{X: x, Y: y}}
		if err := idx.Update(ecs.EntityID(i), bounds); err != nil {
			t.Fatal(err)
		}

	}
	results := idx.Query(geom.Rect{Min: geom.Vector2{X: 50, Y: 25}, Max: geom.Vector2{X: 100, Y: 45}})
	var want []ecs.EntityID
	for row := 0; row < 10; row++ {
		for col := 10; col < 30; col++ {
			want = append(want, ecs.EntityID(row*100+col))
		}
	}
	slices.Sort(results)
	if !slices.Equal(results, want) {
		t.Errorf("Query() = %v, want %v", results, want)
	}
	if len(idx.entities) != 1000 {
		t.Errorf("index has %d entities, want 1000", len(idx.entities))
	}
}

func TestGridUpdatePreservesSharedCells(t *testing.T) {
	idx, _ := NewGrid(10)
	first := geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 15, Y: 15}}
	if err := idx.Update(1, first); err != nil {
		t.Fatal(err)
	}
	entry := idx.entities[1]
	shared := idx.cells[GridCell{}]
	second := geom.Rect{Min: geom.Vector2{X: 1, Y: 1}, Max: geom.Vector2{X: 25, Y: 25}}
	if err := idx.Update(1, second); err != nil {
		t.Fatal(err)
	}
	if idx.entities[1] != entry || &idx.cells[GridCell{}][0] != &shared[0] {
		t.Fatal("shared memberships were replaced")
	}
	if _, exists := idx.cells[GridCell{X: -1, Y: -1}]; exists {
		t.Fatal("old membership remains")
	}
	if !slices.Contains(idx.cells[GridCell{X: 2, Y: 2}], entry) {
		t.Fatal("new membership missing")
	}
	if err := idx.Update(1, geom.Rect{Min: geom.Vector2{X: math.NaN()}}); err == nil {
		t.Fatal("invalid update succeeded")
	}
	if entry.min != (GridCell{}) || entry.max != (GridCell{X: 2, Y: 2}) {
		t.Fatal("invalid update changed memberships")
	}
}

func TestGridSameRangeAndQueriesAllocateNothing(t *testing.T) {
	idx, _ := NewGrid(10)
	bounds := geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 15, Y: 15}}
	if err := idx.Update(1, bounds); err != nil {
		t.Fatal(err)
	}
	buffer := make([]ecs.EntityID, 0, 4)
	if allocs := testing.AllocsPerRun(20, func() {
		if err := idx.Update(1, bounds); err != nil {
			t.Fatal(err)
		}
		buffer = idx.QueryInto(bounds, buffer)
	}); allocs != 0 {
		t.Fatalf("same-range update and query allocated %v times", allocs)
	}
	if len(buffer) != 1 || buffer[0] != 1 {
		t.Fatalf("multi-cell query = %v, want [1]", buffer)
	}
}

func TestGridQueriesMatchCellRangesAfterUpdates(t *testing.T) {
	idx, _ := NewGrid(10)
	rng := rand.New(rand.NewPCG(72, 2))
	bounds := make(map[ecs.EntityID]geom.Rect)
	randomBounds := func() geom.Rect {
		min := geom.Vector2{X: rng.Float64()*100 - 50, Y: rng.Float64()*100 - 50}
		return geom.Rect{Min: min, Max: min.Add(geom.Vector2{X: rng.Float64() * 30, Y: rng.Float64() * 30})}
	}
	for range 200 {
		id := ecs.EntityID(rng.IntN(30))
		if rng.IntN(5) == 0 {
			idx.Remove(id)
			delete(bounds, id)
		} else {
			rect := randomBounds()
			if err := idx.Update(id, rect); err != nil {
				t.Fatal(err)
			}
			bounds[id] = rect
		}
		query := randomBounds()
		qmin, qmax := idx.worldToGrid(query.Min), idx.worldToGrid(query.Max)
		var want, wantOverlaps []ecs.EntityID
		for id, rect := range bounds {
			min, max := idx.worldToGrid(rect.Min), idx.worldToGrid(rect.Max)
			if min.X <= qmax.X && max.X >= qmin.X && min.Y <= qmax.Y && max.Y >= qmin.Y {
				want = append(want, id)
			}
			if rect.OverlapsOrTouches(query) {
				wantOverlaps = append(wantOverlaps, id)
			}
		}
		got := idx.Query(query)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("Query(%v) = %v, want %v", query, got, want)
		}
		gotOverlaps := idx.QueryOverlapsInto(query, nil)
		slices.Sort(gotOverlaps)
		slices.Sort(wantOverlaps)
		if !slices.Equal(gotOverlaps, wantOverlaps) {
			t.Fatalf("QueryOverlapsInto(%v) = %v, want %v", query, gotOverlaps, wantOverlaps)
		}
		for cell, entries := range idx.cells {
			if len(entries) == 0 {
				t.Fatalf("empty cell retained: %v", cell)
			}
		}
	}
}

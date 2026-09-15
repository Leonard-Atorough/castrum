package atlas

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.atlases == nil {
		t.Error("NewStore() returned store with nil atlases map")
	}
}

func TestStoreGetSet(t *testing.T) {
	store := NewStore()

	// Test set and get
	atlas := &Atlas{id: "test"}
	store.set("atlas1", atlas)

	got, ok := store.get("atlas1")
	if !ok {
		t.Fatal("get() returned !ok, want ok")
	}
	if got.id != "test" {
		t.Errorf("get() returned wrong atlas")
	}

	// Test get non-existent
	_, ok = store.get("nonexistent")
	if ok {
		t.Error("get() for non-existent returned ok, want !ok")
	}
}

func TestStoreOverwrite(t *testing.T) {
	store := NewStore()

	atlas1 := &Atlas{id: "first"}
	atlas2 := &Atlas{id: "second"}

	store.set("atlas1", atlas1)
	store.set("atlas1", atlas2)

	got, ok := store.get("atlas1")
	if !ok {
		t.Fatal("get() returned !ok after overwrite")
	}
	if got.id != "second" {
		t.Error("overwrite did not replace value")
	}
}

func TestNewService(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.store != store {
		t.Error("NewService() did not set store correctly")
	}
}

func TestServiceSetGet(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &Atlas{id: "test"}
	svc.Set("atlas1", atlas)

	got, err := svc.Get("atlas1")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.id != "test" {
		t.Error("Get() returned wrong atlas")
	}
}

func TestServiceGetNotFound(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Error("Get() for non-existent error = nil, want error")
	}
}

func TestServiceHas(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &Atlas{id: "test"}
	svc.Set("atlas1", atlas)

	if !svc.Has("atlas1") {
		t.Error("Has() returned false for existing atlas")
	}
	if svc.Has("nonexistent") {
		t.Error("Has() returned true for non-existent atlas")
	}
}

func TestServiceDelete(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &Atlas{id: "test"}
	svc.Set("atlas1", atlas)

	// Verify exists
	if !svc.Has("atlas1") {
		t.Fatal("atlas does not exist before delete")
	}

	// Delete
	svc.Delete("atlas1")

	// Verify deleted
	if svc.Has("atlas1") {
		t.Error("atlas still exists after delete")
	}
}

func TestServiceClear(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas1 := &Atlas{id: "test1"}
	atlas2 := &Atlas{id: "test2"}
	svc.Set("atlas1", atlas1)
	svc.Set("atlas2", atlas2)

	// Verify both exist
	if !svc.Has("atlas1") || !svc.Has("atlas2") {
		t.Fatal("atlases do not exist before clear")
	}

	// Clear
	svc.Clear()

	// Verify both deleted
	if svc.Has("atlas1") || svc.Has("atlas2") {
		t.Error("atlases still exist after clear")
	}
}

func TestServiceDeleteByAssetID(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas1 := &Atlas{id: "test1", assetID: "texture1.png"}
	atlas2 := &Atlas{id: "test2", assetID: "texture1.png"}
	atlas3 := &Atlas{id: "test3", assetID: "texture2.png"}

	// atlas1 and atlas2 use texture1.png, atlas3 uses texture2.png
	svc.Set("atlas1", atlas1)
	svc.Set("atlas2", atlas2)
	svc.Set("atlas3", atlas3)

	// Delete all atlases using texture1.png
	svc.DeleteByAssetID("texture1.png")

	// atlas1 and atlas2 should be gone
	if svc.Has("atlas1") {
		t.Error("atlas1 still exists after DeleteByAssetID")
	}
	if svc.Has("atlas2") {
		t.Error("atlas2 still exists after DeleteByAssetID")
	}

	// atlas3 should still exist
	if !svc.Has("atlas3") {
		t.Error("atlas3 was deleted, but uses different texture")
	}
}

func TestServiceGetSingleflight(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &Atlas{id: "test"}
	svc.Set("atlas1", atlas)

	// Create a channel to coordinate concurrent access
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan any, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := svc.Get("atlas1")
			if err != nil {
				t.Error(err)
				return
			}
			results <- got
		}()
	}

	wg.Wait()
	close(results)

	// All results should be the same atlas instance
	var first any
	count := 0
	for r := range results {
		count++
		if first == nil {
			first = r
		}
		if r != first {
			t.Errorf("concurrent Get() returned different instances")
		}
	}

	if count != numGoroutines {
		t.Errorf("expected %d results, got %d", numGoroutines, count)
	}
}

func TestServiceConcurrentAccess(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	const numGoroutines = 20
	var wg sync.WaitGroup

	// Concurrent sets
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			atlas := &Atlas{id: ID(fmt.Sprintf("atlas_%d", i))}
			svc.Set(fmt.Sprintf("atlas_%d", i), atlas)
		}(i)
	}

	// Concurrent gets
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = svc.Get(fmt.Sprintf("atlas_%d", i))
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, concurrent access is working
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore()

	const numGoroutines = 20
	var wg sync.WaitGroup

	// Concurrent sets
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			atlas := &Atlas{id: ID(fmt.Sprintf("atlas_%d", i))}
			store.set(fmt.Sprintf("atlas_%d", i), atlas)
		}(i)
	}

	// Concurrent gets
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = store.get(fmt.Sprintf("atlas_%d", i))
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, concurrent access is working
}
func TestServiceGetEmptyAtlasID(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	_, err := svc.Get("")
	if err == nil {
		t.Error("Get() with empty atlasID error = nil, want error")
	}
}

func TestServiceDeleteNonExistent(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	// Should not panic
	svc.Delete("nonexistent")
	svc.DeleteByAssetID("nonexistent.png")
}

func TestServiceOverwrite(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas1 := &Atlas{id: "first"}
	atlas2 := &Atlas{id: "second"}

	svc.Set("atlas", atlas1)
	svc.Set("atlas", atlas2)

	got, err := svc.Get("atlas")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.id != "second" {
		t.Error("Set() did not overwrite existing atlas")
	}
}

func TestServiceGetReturnsErrorForNilAtlas(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	err := svc.Set("atlas", nil)
	if err == nil {
		t.Error("Set() with nil atlas error = nil, want error")
	}
}

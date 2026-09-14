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
	atlas := &struct{ ID string }{ID: "test"}
	store.set("atlas1", "texture1.png", atlas)

	got, ok := store.get("atlas1", "texture1.png")
	if !ok {
		t.Fatal("get() returned !ok, want ok")
	}
	if got.(*struct{ ID string }).ID != "test" {
		t.Errorf("get() returned wrong atlas")
	}

	// Test get non-existent
	_, ok = store.get("nonexistent", "texture.png")
	if ok {
		t.Error("get() for non-existent returned ok, want !ok")
	}
}

func TestStoreOverwrite(t *testing.T) {
	store := NewStore()

	atlas1 := &struct{ ID string }{ID: "first"}
	atlas2 := &struct{ ID string }{ID: "second"}

	store.set("atlas1", "texture.png", atlas1)
	store.set("atlas1", "texture.png", atlas2)

	got, ok := store.get("atlas1", "texture.png")
	if !ok {
		t.Fatal("get() returned !ok after overwrite")
	}
	if got.(*struct{ ID string }).ID != "second" {
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

	atlas := &struct{ ID string }{ID: "test"}
	svc.Set("atlas1", "texture1.png", atlas)

	got, err := svc.Get("atlas1", "texture1.png")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.(*struct{ ID string }).ID != "test" {
		t.Error("Get() returned wrong atlas")
	}
}

func TestServiceGetNotFound(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	_, err := svc.Get("nonexistent", "texture.png")
	if err == nil {
		t.Error("Get() for non-existent error = nil, want error")
	}
}

func TestServiceHas(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &struct{ ID string }{ID: "test"}
	svc.Set("atlas1", "texture1.png", atlas)

	if !svc.Has("atlas1", "texture1.png") {
		t.Error("Has() returned false for existing atlas")
	}
	if svc.Has("nonexistent", "texture.png") {
		t.Error("Has() returned true for non-existent atlas")
	}
}

func TestServiceDelete(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &struct{ ID string }{ID: "test"}
	svc.Set("atlas1", "texture1.png", atlas)

	// Verify exists
	if !svc.Has("atlas1", "texture1.png") {
		t.Fatal("atlas does not exist before delete")
	}

	// Delete
	svc.Delete("atlas1", "texture1.png")

	// Verify deleted
	if svc.Has("atlas1", "texture1.png") {
		t.Error("atlas still exists after delete")
	}
}

func TestServiceClear(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas1 := &struct{ ID string }{ID: "test1"}
	atlas2 := &struct{ ID string }{ID: "test2"}
	svc.Set("atlas1", "texture1.png", atlas1)
	svc.Set("atlas2", "texture2.png", atlas2)

	// Verify both exist
	if !svc.Has("atlas1", "texture1.png") || !svc.Has("atlas2", "texture2.png") {
		t.Fatal("atlases do not exist before clear")
	}

	// Clear
	svc.Clear()

	// Verify both deleted
	if svc.Has("atlas1", "texture1.png") || svc.Has("atlas2", "texture2.png") {
		t.Error("atlases still exist after clear")
	}
}

func TestServiceDeleteByAssetID(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas1 := &struct{ ID string }{ID: "test1"}
	atlas2 := &struct{ ID string }{ID: "test2"}
	atlas3 := &struct{ ID string }{ID: "test3"}

	// atlas1 and atlas2 use texture1.png, atlas3 uses texture2.png
	svc.Set("atlas1", "texture1.png", atlas1)
	svc.Set("atlas2", "texture1.png", atlas2)
	svc.Set("atlas3", "texture2.png", atlas3)

	// Delete all atlases using texture1.png
	svc.DeleteByAssetID("texture1.png")

	// atlas1 and atlas2 should be gone
	if svc.Has("atlas1", "texture1.png") {
		t.Error("atlas1 still exists after DeleteByAssetID")
	}
	if svc.Has("atlas2", "texture1.png") {
		t.Error("atlas2 still exists after DeleteByAssetID")
	}

	// atlas3 should still exist
	if !svc.Has("atlas3", "texture2.png") {
		t.Error("atlas3 was deleted, but uses different texture")
	}
}

func TestServiceGetSingleflight(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas := &struct{ ID string }{ID: "test"}
	svc.Set("atlas1", "texture1.png", atlas)

	// Create a channel to coordinate concurrent access
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan any, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := svc.Get("atlas1", "texture1.png")
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
			atlas := &struct{ ID string }{ID: fmt.Sprintf("atlas_%d", i)}
			svc.Set(fmt.Sprintf("atlas_%d", i), "texture.png", atlas)
		}(i)
	}

	// Concurrent gets
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = svc.Get(fmt.Sprintf("atlas_%d", i), "texture.png")
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
			atlas := &struct{ ID string }{ID: fmt.Sprintf("atlas_%d", i)}
			store.set(fmt.Sprintf("atlas_%d", i), "texture.png", atlas)
		}(i)
	}

	// Concurrent gets
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = store.get(fmt.Sprintf("atlas_%d", i), "texture.png")
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, concurrent access is working
}
func TestServiceGetEmptyAtlasID(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	_, err := svc.Get("", "texture.png")
	if err == nil {
		t.Error("Get() with empty atlasID error = nil, want error")
	}
}

func TestServiceGetEmptyAssetID(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	_, err := svc.Get("atlas", "")
	if err == nil {
		t.Error("Get() with empty assetID error = nil, want error")
	}
}

func TestServiceDeleteNonExistent(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	// Should not panic
	svc.Delete("nonexistent", "texture.png")
	svc.DeleteByAssetID("nonexistent.png")
}

func TestServiceOverwrite(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	atlas1 := &struct{ ID string }{ID: "first"}
	atlas2 := &struct{ ID string }{ID: "second"}

	svc.Set("atlas", "texture.png", atlas1)
	svc.Set("atlas", "texture.png", atlas2)

	got, err := svc.Get("atlas", "texture.png")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.(*struct{ ID string }).ID != "second" {
		t.Error("Set() did not overwrite existing atlas")
	}
}

func TestServiceGetReturnsErrorForNilAtlas(t *testing.T) {
	store := NewStore()
	svc := NewService(store)

	err := svc.Set("atlas", "texture.png", nil)
	if err == nil {
		t.Error("Set() with nil atlas error = nil, want error")
	}
}

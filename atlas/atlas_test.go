package atlas

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// mockAtlasStorer is a mock implementation of atlasStorer for testing Builder in isolation
type mockAtlasStorer struct {
	storedAtlas *TextureAtlas
	storedID    string
}

func (m *mockAtlasStorer) store(id string, atlas *TextureAtlas) {
	m.storedID = id
	m.storedAtlas = atlas
}

// createTestTexture creates a test ebiten.Image with the given dimensions
func createTestTexture(width, height int) *ebiten.Image {
	return ebiten.NewImage(width, height)
}

// TestTextureAtlas tests the TextureAtlas struct and its constructor
func TestTextureAtlas(t *testing.T) {
	t.Run("NewTextureAtlas creates a new atlas with correct fields", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		atlas := NewTextureAtlas("test_atlas", texture)

		if atlas == nil {
			t.Error("expected atlas to be non-nil")
		}
		if atlas.ID != "test_atlas" {
			t.Errorf("expected ID 'test_atlas', got %q", atlas.ID)
		}
		if atlas.Texture != texture {
			t.Error("expected Texture to match input texture")
		}
		if atlas.TexW != 256 {
			t.Errorf("expected TexW 256, got %d", atlas.TexW)
		}
		if atlas.TexH != 256 {
			t.Errorf("expected TexH 256, got %d", atlas.TexH)
		}
		if atlas.Regions == nil {
			t.Error("expected Regions map to be initialized")
		}
		if len(atlas.Regions) != 0 {
			t.Errorf("expected empty Regions map, got %d entries", len(atlas.Regions))
		}
	})

	t.Run("NewTextureAtlas with different dimensions", func(t *testing.T) {
		texture := createTestTexture(512, 128)
		atlas := NewTextureAtlas("another_atlas", texture)

		if atlas.TexW != 512 {
			t.Errorf("expected TexW 512, got %d", atlas.TexW)
		}
		if atlas.TexH != 128 {
			t.Errorf("expected TexH 128, got %d", atlas.TexH)
		}
	})
}

// TestBuilder tests the Builder struct and its methods
func TestBuilder(t *testing.T) {
	t.Run("NewBuilder creates a new builder with correct fields", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		if builder == nil {
			t.Error("expected builder to be non-nil")
		}
	})

	t.Run("SliceRegion adds a valid region to the builder", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		builder.SliceRegion("test_region", 0, 0, 32, 32)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(atlas.Regions) != 1 {
			t.Errorf("expected 1 region, got %d", len(atlas.Regions))
		}

		region := atlas.Regions["test_region"]
		if region == nil {
			t.Error("expected region 'test_region' to exist")
		}
		if region.Name != "test_region" {
			t.Errorf("expected region name 'test_region', got %q", region.Name)
		}
		if region.Width != 32 {
			t.Errorf("expected region width 32, got %d", region.Width)
		}
		if region.Height != 32 {
			t.Errorf("expected region height 32, got %d", region.Height)
		}
		if region.Image == nil {
			t.Error("expected region Image to be non-nil")
		}
	})

	t.Run("SliceRegion silently skips out-of-bounds regions", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		// Add a valid region first
		builder.SliceRegion("valid_region", 0, 0, 32, 32)
		// Add an out-of-bounds region (x + width > texture width)
		builder.SliceRegion("out_of_bounds", 200, 200, 100, 100)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(atlas.Regions) != 1 {
			t.Errorf("expected 1 region (out-of-bounds skipped), got %d", len(atlas.Regions))
		}
		if _, ok := atlas.Regions["valid_region"]; !ok {
			t.Error("expected valid_region to exist")
		}
		if _, ok := atlas.Regions["out_of_bounds"]; ok {
			t.Error("expected out_of_bounds to be skipped")
		}
	})

	t.Run("SliceRegion with negative dimensions is skipped", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		// Add a valid region first
		builder.SliceRegion("valid_region", 0, 0, 32, 32)
		// Negative width/height will create an invalid rectangle
		builder.SliceRegion("negative_dims", 0, 0, -32, -32)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(atlas.Regions) != 1 {
			t.Errorf("expected 1 region (negative dims skipped), got %d", len(atlas.Regions))
		}
		if _, ok := atlas.Regions["valid_region"]; !ok {
			t.Error("expected valid_region to exist")
		}
		if _, ok := atlas.Regions["negative_dims"]; ok {
			t.Error("expected negative_dims to be skipped")
		}
	})

	t.Run("AutoSlice creates regions in a grid", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		// Create a 4x4 grid with 64x64 frames
		builder.AutoSlice(4, 4, 64, 64, nil)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(atlas.Regions) != 16 {
			t.Errorf("expected 16 regions (4x4 grid), got %d", len(atlas.Regions))
		}

		// Check that regions are named with default naming (0, 1, 2, ...)
		for i := 0; i < 16; i++ {
			name := fmt.Sprintf("%d", i)
			if _, ok := atlas.Regions[name]; !ok {
				t.Errorf("expected region %q to exist", name)
			}
		}
	})

	t.Run("AutoSlice with custom name function", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		// Create a 2x2 grid with custom names
		builder.AutoSlice(2, 2, 128, 128, func(idx int) string {
			return fmt.Sprintf("frame_%d", idx)
		})

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(atlas.Regions) != 4 {
			t.Errorf("expected 4 regions (2x2 grid), got %d", len(atlas.Regions))
		}

		// Check custom names
		expectedNames := []string{"frame_0", "frame_1", "frame_2", "frame_3"}
		for _, name := range expectedNames {
			if _, ok := atlas.Regions[name]; !ok {
				t.Errorf("expected region %q to exist", name)
			}
		}
	})

	t.Run("Build returns error if texture is nil", func(t *testing.T) {
		mockStore := &mockAtlasStorer{}
		builder := &Builder{
			id:      "test_builder",
			texture: nil,
			regions: make(map[string]*SubTexture),
			store:   mockStore,
		}

		atlas, err := builder.Build()
		if err == nil {
			t.Error("expected error for nil texture, got nil")
		}
		if atlas != nil {
			t.Error("expected atlas to be nil when texture is nil")
		}
		// Check error message contains expected substring
		expectedSubstring := "texture is required"
		if !contains(err.Error(), expectedSubstring) {
			t.Errorf("expected error to contain %q, got %q", expectedSubstring, err.Error())
		}
	})

	t.Run("Build returns error if no regions are added", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		atlas, err := builder.Build()
		if err == nil {
			t.Error("expected error for no regions, got nil")
		}
		if atlas != nil {
			t.Error("expected atlas to be nil when no regions are added")
		}
	})

	t.Run("Build stores atlas with correct ID and fields", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		builder.SliceRegion("test_region", 0, 0, 32, 32)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if mockStore.storedID != "test_builder" {
			t.Errorf("expected stored ID 'test_builder', got %q", mockStore.storedID)
		}
		if mockStore.storedAtlas != atlas {
			t.Error("expected stored atlas to match built atlas")
		}
		if atlas.ID != "test_builder" {
			t.Errorf("expected atlas ID 'test_builder', got %q", atlas.ID)
		}
		if atlas.Texture != texture {
			t.Error("expected atlas Texture to match input texture")
		}
	})

	t.Run("Build with multiple regions", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		builder.SliceRegion("region1", 0, 0, 32, 32)
		builder.SliceRegion("region2", 32, 0, 32, 32)
		builder.SliceRegion("region3", 64, 0, 32, 32)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(atlas.Regions) != 3 {
			t.Errorf("expected 3 regions, got %d", len(atlas.Regions))
		}
	})

	t.Run("SliceRegion can be chained", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		result := builder.SliceRegion("region1", 0, 0, 32, 32).SliceRegion("region2", 32, 0, 32, 32)

		if result != builder {
			t.Error("expected SliceRegion to return the same builder for chaining")
		}
	})

	t.Run("AutoSlice can be chained", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		mockStore := &mockAtlasStorer{}
		builder := NewBuilder("test_builder", texture, mockStore)

		result := builder.AutoSlice(2, 2, 128, 128, nil)

		if result != builder {
			t.Error("expected AutoSlice to return the same builder for chaining")
		}
	})
}

// TestAtlasStore tests the AtlasStore struct and its methods
func TestAtlasStore(t *testing.T) {
	t.Run("NewAtlasStore creates a new store", func(t *testing.T) {
		store := NewAtlasStore()
		if store == nil {
			t.Error("expected store to be non-nil")
		}
	})

	t.Run("NewAtlas creates a new builder", func(t *testing.T) {
		store := NewAtlasStore()
		builder := store.NewAtlas("test_atlas")

		if builder == nil {
			t.Error("expected builder to be non-nil")
		}
	})

	t.Run("NewAtlas initializes builder with correct store", func(t *testing.T) {
		store := NewAtlasStore()
		builder := store.NewAtlas("test_atlas")

		// Build a test atlas to verify the store is correctly set
		texture := createTestTexture(256, 256)
		builder.texture = texture
		builder.SliceRegion("test_region", 0, 0, 32, 32)

		atlas, err := builder.Build()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify the atlas was stored
		retrieved := store.Get("test_atlas")
		if retrieved == nil {
			t.Error("expected atlas to be stored")
		}
		if retrieved != atlas {
			t.Error("expected retrieved atlas to match built atlas")
		}
	})

	t.Run("Get retrieves a stored atlas", func(t *testing.T) {
		store := NewAtlasStore()
		atlas := &TextureAtlas{
			ID:      "test_atlas",
			Texture: createTestTexture(256, 256),
			TexW:    256,
			TexH:    256,
			Regions: make(map[string]*SubTexture),
		}

		store.store("test_atlas", atlas)

		retrieved := store.Get("test_atlas")
		if retrieved == nil {
			t.Error("expected atlas to be retrieved")
		}
		if retrieved != atlas {
			t.Error("expected retrieved atlas to match stored atlas")
		}
	})

	t.Run("Get returns nil for non-existent atlas", func(t *testing.T) {
		store := NewAtlasStore()

		retrieved := store.Get("nonexistent")
		if retrieved != nil {
			t.Error("expected nil for non-existent atlas")
		}
	})

	t.Run("Has returns true for existing atlas", func(t *testing.T) {
		store := NewAtlasStore()
		atlas := &TextureAtlas{ID: "test_atlas"}
		store.store("test_atlas", atlas)

		if !store.Has("test_atlas") {
			t.Error("expected Has to return true for existing atlas")
		}
	})

	t.Run("Has returns false for non-existent atlas", func(t *testing.T) {
		store := NewAtlasStore()

		if store.Has("nonexistent") {
			t.Error("expected Has to return false for non-existent atlas")
		}
	})

	t.Run("Remove deletes an atlas", func(t *testing.T) {
		store := NewAtlasStore()
		atlas := &TextureAtlas{ID: "test_atlas"}
		store.store("test_atlas", atlas)

		store.Remove("test_atlas")

		if store.Has("test_atlas") {
			t.Error("expected atlas to be removed")
		}
		if store.Get("test_atlas") != nil {
			t.Error("expected Get to return nil after removal")
		}
	})

	t.Run("Remove does not panic for non-existent atlas", func(t *testing.T) {
		store := NewAtlasStore()
		// Should not panic
		store.Remove("nonexistent")
	})

	t.Run("Store and retrieve multiple atlases", func(t *testing.T) {
		store := NewAtlasStore()

		atlas1 := &TextureAtlas{ID: "atlas1"}
		atlas2 := &TextureAtlas{ID: "atlas2"}
		atlas3 := &TextureAtlas{ID: "atlas3"}

		store.store("atlas1", atlas1)
		store.store("atlas2", atlas2)
		store.store("atlas3", atlas3)

		if !store.Has("atlas1") || !store.Has("atlas2") || !store.Has("atlas3") {
			t.Error("expected all atlases to be stored")
		}

		if store.Get("atlas1") != atlas1 {
			t.Error("expected to retrieve atlas1")
		}
		if store.Get("atlas2") != atlas2 {
			t.Error("expected to retrieve atlas2")
		}
		if store.Get("atlas3") != atlas3 {
			t.Error("expected to retrieve atlas3")
		}
	})

	t.Run("Overwrite existing atlas", func(t *testing.T) {
		store := NewAtlasStore()
		atlas1 := &TextureAtlas{ID: "test_atlas"}
		atlas2 := &TextureAtlas{ID: "test_atlas"}

		store.store("test_atlas", atlas1)
		store.store("test_atlas", atlas2)

		retrieved := store.Get("test_atlas")
		if retrieved != atlas2 {
			t.Error("expected atlas to be overwritten")
		}
	})

	t.Run("Concurrent store and get", func(t *testing.T) {
		store := NewAtlasStore()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				atlas := &TextureAtlas{
					ID:      fmt.Sprintf("atlas_%d", id),
					Texture: createTestTexture(32, 32),
					TexW:    32,
					TexH:    32,
					Regions: make(map[string]*SubTexture),
				}
				store.store(fmt.Sprintf("atlas_%d", id), atlas)

				retrieved := store.Get(fmt.Sprintf("atlas_%d", id))
				if retrieved == nil {
					t.Error("failed to retrieve atlas concurrently")
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Concurrent Has and Remove", func(t *testing.T) {
		store := NewAtlasStore()

		// Pre-populate some atlases
		for i := 0; i < 50; i++ {
			atlas := &TextureAtlas{ID: fmt.Sprintf("atlas_%d", i)}
			store.store(fmt.Sprintf("atlas_%d", i), atlas)
		}

		var wg sync.WaitGroup

		// Concurrent Has checks
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if !store.Has(fmt.Sprintf("atlas_%d", id)) {
					t.Error("expected atlas to exist")
				}
			}(i)
		}

		// Concurrent Remove
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				store.Remove(fmt.Sprintf("atlas_%d", id))
			}(i)
		}

		wg.Wait()
	})
}

// TestSubTexture tests the SubTexture struct
func TestSubTexture(t *testing.T) {
	t.Run("SubTexture has correct fields", func(t *testing.T) {
		texture := createTestTexture(256, 256)
		subTexture := &SubTexture{
			Name:   "test_region",
			Image:  texture.SubImage(image.Rect(0, 0, 32, 32)).(*ebiten.Image),
			Width:  32,
			Height: 32,
		}

		if subTexture.Name != "test_region" {
			t.Errorf("expected Name 'test_region', got %q", subTexture.Name)
		}
		if subTexture.Width != 32 {
			t.Errorf("expected Width 32, got %d", subTexture.Width)
		}
		if subTexture.Height != 32 {
			t.Errorf("expected Height 32, got %d", subTexture.Height)
		}
		if subTexture.Image == nil {
			t.Error("expected Image to be non-nil")
		}
	})
}

// Note: FromMetadata is harder to test without creating actual JSON files in a test filesystem.
// For now, we'll skip it, but in a real project, you might want to:
// 1. Create a temporary filesystem with a test JSON file
// 2. Test successful metadata loading
// 3. Test error cases (invalid JSON, missing file, etc.)

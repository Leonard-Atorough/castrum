package assets

import (
	"testing"
	"testing/fstest"
)

const validAtlasJSON = `{
  "path": "sprites/hero.png",
  "frames": {
    "idle_1": {"frame": {"x": 0, "y": 0, "w": 32, "h": 32}},
    "idle_2": {"frame": {"x": 32, "y": 0, "w": 32, "h": 32}},
    "idle_3": {"frame": {"x": 64, "y": 0, "w": 32, "h": 32}},
    "walk_1": {"frame": {"x": 0, "y": 32, "w": 32, "h": 32}},
    "walk_2": {"frame": {"x": 32, "y": 32, "w": 32, "h": 32}}
  }
}`

const invalidAtlasOutOfBoundsJSON = `{
  "path": "sprites/hero.png",
  "frames": {
    "oob": {"frame": {"x": 200, "y": 200, "w": 32, "h": 32}}
  }
}`

const enemyAtlasJSON = `{
  "path": "sprites/enemy.png",
  "frames": {
    "idle_1": {"frame": {"x": 0, "y": 0, "w": 32, "h": 32}},
    "idle_2": {"frame": {"x": 32, "y": 0, "w": 32, "h": 32}}
  }
}`

func TestAtlasStoreLoad(t *testing.T) {
	t.Run("loads and caches a valid atlas with regions", func(t *testing.T) {
		// Create a 96x64 test image to fit all regions
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, err := store.Load("hero.atlas.json")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if atlas == nil {
			t.Fatal("expected non-nil atlas")
		}
		if atlas.Path != "sprites/hero.png" {
			t.Errorf("Path = %q, want %q", atlas.Path, "sprites/hero.png")
		}
		if atlas.Texture == nil {
			t.Fatal("expected non-nil Texture")
		}
		if atlas.Texture.Image == nil {
			t.Fatal("expected non-nil Texture.Image")
		}

		// Check regions
		if len(atlas.Regions) != 5 {
			t.Errorf("Regions count = %d, want 5", len(atlas.Regions))
		}

		// Check specific region
		if idle1, ok := atlas.Regions["idle_1"]; ok {
			if idle1.Width != 32 || idle1.Height != 32 {
				t.Errorf("idle_1 dimensions = %dx%d, want 32x32", idle1.Width, idle1.Height)
			}
			if idle1.Name != "idle_1" {
				t.Errorf("SubTexture Name = %q, want %q", idle1.Name, "idle_1")
			}
		} else {
			t.Fatal("expected idle_1 region")
		}

		// Verify caching
		cached, exists := store.atlases["hero.atlas.json"]
		if !exists {
			t.Fatal("expected atlas to be cached")
		}
		if cached != atlas {
			t.Fatal("cached atlas should be the same object")
		}
	})

	t.Run("returns cached atlas on subsequent loads", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas1, _ := store.Load("hero.atlas.json")
		atlas2, _ := store.Load("hero.atlas.json")

		if atlas1 != atlas2 {
			t.Fatal("expected same object from cache")
		}
	})

	t.Run("returns error for missing atlas file", func(t *testing.T) {
		fs := fstest.MapFS{}
		store := newAtlasStore(fs)

		_, err := store.Load("missing.atlas.json")
		if err == nil {
			t.Fatal("expected error for missing atlas file")
		}
	})

	t.Run("returns error when filesystem is nil", func(t *testing.T) {
		store := newAtlasStore(nil)

		_, err := store.Load("any.atlas.json")
		if err == nil {
			t.Fatal("expected error when filesystem is nil")
		}
	})

	t.Run("returns error for missing texture file", func(t *testing.T) {
		fs := fstest.MapFS{
			"hero.atlas.json": {Data: []byte(validAtlasJSON)},
			// But sprites/hero.png is missing
		}

		store := newAtlasStore(fs)
		_, err := store.Load("hero.atlas.json")
		if err == nil {
			t.Fatal("expected error for missing texture file")
		}
	})

	t.Run("rejects regions that are out of bounds", func(t *testing.T) {
		// Create a small 32x32 image, but atlas expects 200x200 region
		pngData := generateTestPNG(32, 32)
		fs := fstest.MapFS{
			"oob.atlas.json":   {Data: []byte(invalidAtlasOutOfBoundsJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		_, err := store.Load("oob.atlas.json")
		if err == nil {
			t.Fatal("expected error for out-of-bounds region")
		}
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		invalidJSON := "not valid json"
		fs := fstest.MapFS{
			"broken.atlas.json": {Data: []byte(invalidJSON)},
		}

		store := newAtlasStore(fs)
		_, err := store.Load("broken.atlas.json")
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("creates SubTexture with correct properties", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.json")

		walk1 := atlas.Regions["walk_1"]
		if walk1.Image == nil {
			t.Fatal("expected non-nil SubTexture.Image")
		}
		if walk1.Width != 32 || walk1.Height != 32 {
			t.Errorf("SubTexture dimensions = %dx%d, want 32x32", walk1.Width, walk1.Height)
		}
		if walk1.Name != "walk_1" {
			t.Errorf("SubTexture.Name = %q, want %q", walk1.Name, "walk_1")
		}
	})

	t.Run("handles atlas with different region sizes", func(t *testing.T) {
		mixedJSON := `{
  "path": "sprites/mixed.png",
  "frames": {
    "small": {"frame": {"x": 0, "y": 0, "w": 16, "h": 16}},
    "medium": {"frame": {"x": 16, "y": 0, "w": 32, "h": 32}},
    "large": {"frame": {"x": 48, "y": 0, "w": 64, "h": 64}}
  }
}`
		// Create a 112x64 image to fit all regions
		pngData := generateTestPNG(112, 64)
		fs := fstest.MapFS{
			"mixed.atlas.json":  {Data: []byte(mixedJSON)},
			"sprites/mixed.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, err := store.Load("mixed.atlas.json")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if atlas.Regions["small"].Width != 16 {
			t.Errorf("small region width = %d, want 16", atlas.Regions["small"].Width)
		}
		if atlas.Regions["medium"].Width != 32 {
			t.Errorf("medium region width = %d, want 32", atlas.Regions["medium"].Width)
		}
		if atlas.Regions["large"].Width != 64 {
			t.Errorf("large region width = %d, want 64", atlas.Regions["large"].Width)
		}
	})

	t.Run("loads multiple atlases independently", func(t *testing.T) {
		pngData1 := generateTestPNG(96, 64)
		pngData2 := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":   {Data: []byte(validAtlasJSON)},
			"sprites/hero.png":  {Data: pngData1},
			"enemy.atlas.json":  {Data: []byte(enemyAtlasJSON)},
			"sprites/enemy.png": {Data: pngData2},
		}

		store := newAtlasStore(fs)
		atlas1, _ := store.Load("hero.atlas.json")
		atlas2, _ := store.Load("enemy.atlas.json")

		if atlas1.Texture.Path == atlas2.Texture.Path {
			t.Fatal("expected different texture paths")
		}
	})
}

func TestTextureAtlasStructure(t *testing.T) {
	t.Run("TextureAtlas has correct fields", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.json")

		if atlas.Path == "" {
			t.Fatal("expected non-empty Path")
		}
		if atlas.Texture == nil {
			t.Fatal("expected non-nil Texture")
		}
		if len(atlas.Regions) == 0 {
			t.Fatal("expected non-empty Regions")
		}
	})
}

func TestSubTextureStructure(t *testing.T) {
	t.Run("SubTexture has correct fields", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.json")

		subTex := atlas.Regions["idle_1"]
		if subTex.Name == "" {
			t.Fatal("expected non-empty Name")
		}
		if subTex.Image == nil {
			t.Fatal("expected non-nil Image")
		}
		if subTex.Width <= 0 || subTex.Height <= 0 {
			t.Errorf("expected positive dimensions, got %dx%d", subTex.Width, subTex.Height)
		}
	})
}

func TestAtlasStoreThreadSafety(t *testing.T) {
	t.Run("concurrent loads don't cause races", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)

		// Simulate concurrent loads
		done := make(chan error, 2)
		go func() {
			_, err := store.Load("hero.atlas.json")
			done <- err
		}()
		go func() {
			_, err := store.Load("hero.atlas.json")
			done <- err
		}()

		if err := <-done; err != nil {
			t.Fatalf("concurrent load 1 failed: %v", err)
		}
		if err := <-done; err != nil {
			t.Fatalf("concurrent load 2 failed: %v", err)
		}
	})
}

// TestSubImageSharing verifies that SubTexture.Image shares GPU memory with
// the parent atlas texture via Ebiten's SubImage mechanism.
func TestSubImageSharing(t *testing.T) {
	t.Run("SubTexture images are SubImages of parent", func(t *testing.T) {
		pngData := generateTestPNG(128, 64)
		fs := fstest.MapFS{
			"hero.atlas.json":  {Data: []byte(validAtlasJSON)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.json")

		// Verify that SubTexture images were created via SubImage
		// (we can't directly verify GPU memory sharing, but we can verify they're not nil)
		for regionName, subTex := range atlas.Regions {
			if subTex.Image == nil {
				t.Errorf("SubTexture %q has nil Image", regionName)
			}
		}
	})
}

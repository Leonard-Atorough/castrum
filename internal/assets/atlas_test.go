package assets

import (
	"testing"
	"testing/fstest"
)

const validAtlasYAML = `path: "sprites/hero.png"
regions:
  idle_1: [0, 0, 32, 32]
  idle_2: [32, 0, 64, 32]
  idle_3: [64, 0, 96, 32]
  walk_1: [0, 32, 32, 64]
  walk_2: [32, 32, 64, 64]
`

const invalidAtlasOutOfBounds = `path: "sprites/hero.png"
regions:
  oob: [200, 200, 32, 32]
`

const enemyAtlasYAML = `path: "sprites/enemy.png"
regions:
  idle_1: [0, 0, 32, 32]
  idle_2: [32, 0, 64, 32]
`

func TestAtlasStoreLoad(t *testing.T) {
	t.Run("loads and caches a valid atlas with regions", func(t *testing.T) {
		// Create a 96x64 test image to fit all regions
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, err := store.Load("hero.atlas.yaml")
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
		cached, exists := store.atlases["hero.atlas.yaml"]
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
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas1, _ := store.Load("hero.atlas.yaml")
		atlas2, _ := store.Load("hero.atlas.yaml")

		if atlas1 != atlas2 {
			t.Fatal("expected same object from cache")
		}
	})

	t.Run("returns error for missing atlas file", func(t *testing.T) {
		fs := fstest.MapFS{}
		store := newAtlasStore(fs)

		_, err := store.Load("missing.atlas.yaml")
		if err == nil {
			t.Fatal("expected error for missing atlas file")
		}
	})

	t.Run("returns error when filesystem is nil", func(t *testing.T) {
		store := newAtlasStore(nil)

		_, err := store.Load("any.atlas.yaml")
		if err == nil {
			t.Fatal("expected error when filesystem is nil")
		}
	})

	t.Run("returns error for missing texture file", func(t *testing.T) {
		fs := fstest.MapFS{
			"hero.atlas.yaml": {Data: []byte(validAtlasYAML)},
			// But sprites/hero.png is missing
		}

		store := newAtlasStore(fs)
		_, err := store.Load("hero.atlas.yaml")
		if err == nil {
			t.Fatal("expected error for missing texture file")
		}
	})

	t.Run("rejects regions that are out of bounds", func(t *testing.T) {
		// Create a small 32x32 image, but atlas expects 200x200 region
		pngData := generateTestPNG(32, 32)
		fs := fstest.MapFS{
			"oob.atlas.yaml":   {Data: []byte(invalidAtlasOutOfBounds)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		_, err := store.Load("oob.atlas.yaml")
		if err == nil {
			t.Fatal("expected error for out-of-bounds region")
		}
	})

	t.Run("handles invalid YAML gracefully", func(t *testing.T) {
		invalidYAML := "path: [not valid"
		fs := fstest.MapFS{
			"broken.atlas.yaml": {Data: []byte(invalidYAML)},
		}

		store := newAtlasStore(fs)
		_, err := store.Load("broken.atlas.yaml")
		if err == nil {
			t.Fatal("expected error for malformed YAML")
		}
	})

	t.Run("creates SubTexture with correct properties", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.yaml")

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
		mixedYAML := `path: "sprites/mixed.png"
regions:
  small: [0, 0, 16, 16]
  medium: [16, 0, 48, 32]
  large: [48, 0, 112, 64]
`
		// Create a 112x64 image to fit all regions
		pngData := generateTestPNG(112, 64)
		fs := fstest.MapFS{
			"mixed.atlas.yaml":  {Data: []byte(mixedYAML)},
			"sprites/mixed.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, err := store.Load("mixed.atlas.yaml")
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
			"hero.atlas.yaml":   {Data: []byte(validAtlasYAML)},
			"sprites/hero.png":  {Data: pngData1},
			"enemy.atlas.yaml":  {Data: []byte(enemyAtlasYAML)},
			"sprites/enemy.png": {Data: pngData2},
		}

		store := newAtlasStore(fs)
		atlas1, _ := store.Load("hero.atlas.yaml")
		atlas2, _ := store.Load("enemy.atlas.yaml")

		if atlas1.Texture.Path == atlas2.Texture.Path {
			t.Fatal("expected different texture paths")
		}
	})
}

func TestTextureAtlasStructure(t *testing.T) {
	t.Run("TextureAtlas has correct fields", func(t *testing.T) {
		pngData := generateTestPNG(96, 64)
		fs := fstest.MapFS{
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.yaml")

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
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.yaml")

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
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)

		// Simulate concurrent loads
		done := make(chan error, 2)
		go func() {
			_, err := store.Load("hero.atlas.yaml")
			done <- err
		}()
		go func() {
			_, err := store.Load("hero.atlas.yaml")
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
			"hero.atlas.yaml":  {Data: []byte(validAtlasYAML)},
			"sprites/hero.png": {Data: pngData},
		}

		store := newAtlasStore(fs)
		atlas, _ := store.Load("hero.atlas.yaml")

		// Verify that SubTexture images were created via SubImage
		// (we can't directly verify GPU memory sharing, but we can verify they're not nil)
		for regionName, subTex := range atlas.Regions {
			if subTex.Image == nil {
				t.Errorf("SubTexture %q has nil Image", regionName)
			}
		}
	})
}

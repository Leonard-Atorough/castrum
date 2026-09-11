package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"
)

// generateTestPNG creates a minimal PNG image for testing.
// Returns the PNG bytes as a slice.
func generateTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a solid color for simplicity
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // Should never happen in tests
	}
	return buf.Bytes()
}

func TestTextureStoreLoad(t *testing.T) {
	t.Run("loads and caches a texture by path", func(t *testing.T) {
		pngData := generateTestPNG(64, 64)
		fs := fstest.MapFS{
			"sprite.png": {Data: pngData},
		}

		store := newTextureStore(fs)
		tex, err := store.Load("sprite.png")
		if err != nil {
			t.Errorf("Load failed: %v", err)
		}
		if tex == nil {
			t.Error("expected non-nil texture")
		}
		if tex.Path != "sprite.png" {
			t.Errorf("Path = %q, want %q", tex.Path, "sprite.png")
		}
		if tex.Width != 64 || tex.Height != 64 {
			t.Errorf("dimensions = %dx%d, want 64x64", tex.Width, tex.Height)
		}
		if tex.Image == nil {
			t.Error("expected non-nil Image")
		}

		// Verify caching by checking it's in the store
		cached, exists := store.Textures["sprite.png"]
		if !exists {
			t.Error("expected texture to be cached")
		}
		if cached != tex {
			t.Error("cached texture should be the same object")
		}
	})

	t.Run("returns cached texture on subsequent loads", func(t *testing.T) {
		pngData := generateTestPNG(32, 32)
		fs := fstest.MapFS{
			"cached.png": {Data: pngData},
		}

		store := newTextureStore(fs)
		tex1, _ := store.Load("cached.png")
		tex2, _ := store.Load("cached.png")

		if tex1 != tex2 {
			t.Error("expected same object from cache")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		fs := fstest.MapFS{}
		store := newTextureStore(fs)

		_, err := store.Load("missing.png")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("defaults to current directory when filesystem is nil", func(t *testing.T) {
		store := newTextureStore(nil)

		// Since the current directory is unlikely to have "sprite.png", we expect an error
		_, err := store.Load("sprite.png")
		if err == nil {
			t.Error("expected error for missing file in default filesystem")
		}
	})

	t.Run("handles jpg format", func(t *testing.T) {
		// For jpg, we need to use a different approach. For now, we'll just test
		// that the extension is recognized by the broader asset system.
		// Actual jpg encoding is more complex and less common in tests.
		t.Skip("jpg test requires external image encoding")
	})

	t.Run("loads multiple textures independently", func(t *testing.T) {
		pngData1 := generateTestPNG(64, 64)
		pngData2 := generateTestPNG(32, 32)
		fs := fstest.MapFS{
			"sprite1.png": {Data: pngData1},
			"sprite2.png": {Data: pngData2},
		}

		store := newTextureStore(fs)
		tex1, _ := store.Load("sprite1.png")
		tex2, _ := store.Load("sprite2.png")

		if tex1.Path == tex2.Path {
			t.Error("expected different paths")
		}
		if tex1.Width == tex2.Width {
			t.Errorf("expected different widths: %d vs %d", tex1.Width, tex2.Width)
		}
	})
}

func TestTextureStructure(t *testing.T) {
	t.Run("Texture has correct fields", func(t *testing.T) {
		pngData := generateTestPNG(16, 16)
		fs := fstest.MapFS{
			"test.png": {Data: pngData},
		}

		store := newTextureStore(fs)
		tex, _ := store.Load("test.png")

		if tex.Path == "" {
			t.Error("expected non-empty Path")
		}
		if tex.Image == nil {
			t.Error("expected non-nil Image")
		}
		if tex.Width <= 0 || tex.Height <= 0 {
			t.Errorf("expected positive dimensions, got %dx%d", tex.Width, tex.Height)
		}
	})
}

func TestTextureStoreThreadSafety(t *testing.T) {
	t.Run("concurrent loads don't cause races", func(t *testing.T) {
		pngData := generateTestPNG(16, 16)
		fs := fstest.MapFS{
			"concurrent.png": {Data: pngData},
		}

		store := newTextureStore(fs)

		// Simulate concurrent loads
		done := make(chan error, 2)
		go func() {
			_, err := store.Load("concurrent.png")
			done <- err
		}()
		go func() {
			_, err := store.Load("concurrent.png")
			done <- err
		}()

		if err := <-done; err != nil {
			t.Errorf("concurrent load 1 failed: %v", err)
		}
		if err := <-done; err != nil {
			t.Errorf("concurrent load 2 failed: %v", err)
		}
	})
}

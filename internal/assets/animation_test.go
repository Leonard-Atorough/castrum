package assets

import (
	"testing"
	"testing/fstest"
)

const validAnimationYAML = `frames:
  - "frame1.png"
  - "frame2.png"
  - "frame3.png"
frameSpeed: 0.1
loop: true
`

const validAtlasAnimationYAML = `atlasPath: "sprites/hero.atlas.yaml"
frames:
  - "idle_1"
  - "idle_2"
  - "idle_3"
frameSpeed: 0.15
loop: false
`

const invalidAnimationEmptyFramesYAML = `frames: []
frameSpeed: 0.1
loop: true
`

const invalidAnimationZeroSpeedYAML = `frames:
  - "frame1.png"
frameSpeed: 0
loop: true
`

const invalidAnimationNegativeSpeedYAML = `frames:
  - "frame1.png"
frameSpeed: -0.1
loop: true
`

func TestAnimationStoreLoad(t *testing.T) {
	t.Run("loads and caches a valid animation clip", func(t *testing.T) {
		fs := fstest.MapFS{
			"anim.anim.yaml": {Data: []byte(validAnimationYAML)},
		}

		store := newAnimationStore(fs)
		clip, err := store.Load("anim.anim.yaml")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if clip == nil {
			t.Fatal("expected non-nil clip")
		}
		if len(clip.Frames) != 3 {
			t.Errorf("Frames count = %d, want 3", len(clip.Frames))
		}
		if clip.FrameSpeed != 0.1 {
			t.Errorf("FrameSpeed = %v, want 0.1", clip.FrameSpeed)
		}
		if !clip.Loop {
			t.Errorf("Loop = %v, want true", clip.Loop)
		}

		// Verify caching
		cached, exists := store.Animations["anim.anim.yaml"]
		if !exists {
			t.Fatal("expected animation to be cached")
		}
		if cached != clip {
			t.Fatal("cached clip should be the same object")
		}
	})

	t.Run("loads animation with atlas path", func(t *testing.T) {
		fs := fstest.MapFS{
			"hero.anim.yaml": {Data: []byte(validAtlasAnimationYAML)},
		}

		store := newAnimationStore(fs)
		clip, err := store.Load("hero.anim.yaml")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if clip.AtlasPath != "sprites/hero.atlas.yaml" {
			t.Errorf("AtlasPath = %q, want %q", clip.AtlasPath, "sprites/hero.atlas.yaml")
		}
		if len(clip.Frames) != 3 {
			t.Errorf("Frames = %d, want 3", len(clip.Frames))
		}
		// When AtlasPath is set, Frames should be frame names, not paths
		if clip.Frames[0] != "idle_1" {
			t.Errorf("first frame = %q, want 'idle_1'", clip.Frames[0])
		}
	})

	t.Run("returns cached clip on subsequent loads", func(t *testing.T) {
		fs := fstest.MapFS{
			"anim.anim.yaml": {Data: []byte(validAnimationYAML)},
		}

		store := newAnimationStore(fs)
		clip1, _ := store.Load("anim.anim.yaml")
		clip2, _ := store.Load("anim.anim.yaml")

		if clip1 != clip2 {
			t.Fatal("expected same object from cache")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		fs := fstest.MapFS{}
		store := newAnimationStore(fs)

		_, err := store.Load("missing.anim.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("rejects empty frames list", func(t *testing.T) {
		fs := fstest.MapFS{
			"empty.anim.yaml": {Data: []byte(invalidAnimationEmptyFramesYAML)},
		}

		store := newAnimationStore(fs)
		_, err := store.Load("empty.anim.yaml")
		if err == nil {
			t.Fatal("expected error for empty frames")
		}
	})

	t.Run("rejects zero frame speed", func(t *testing.T) {
		fs := fstest.MapFS{
			"zero.anim.yaml": {Data: []byte(invalidAnimationZeroSpeedYAML)},
		}

		store := newAnimationStore(fs)
		_, err := store.Load("zero.anim.yaml")
		if err == nil {
			t.Fatal("expected error for zero frame speed")
		}
	})

	t.Run("rejects negative frame speed", func(t *testing.T) {
		fs := fstest.MapFS{
			"negative.anim.yaml": {Data: []byte(invalidAnimationNegativeSpeedYAML)},
		}

		store := newAnimationStore(fs)
		_, err := store.Load("negative.anim.yaml")
		if err == nil {
			t.Fatal("expected error for negative frame speed")
		}
	})

	t.Run("handles invalid YAML gracefully", func(t *testing.T) {
		invalidYAML := "not: [valid yaml"
		fs := fstest.MapFS{
			"broken.anim.yaml": {Data: []byte(invalidYAML)},
		}

		store := newAnimationStore(fs)
		_, err := store.Load("broken.anim.yaml")
		if err == nil {
			t.Fatal("expected error for malformed YAML")
		}
	})

	t.Run("allows loop to be optional (defaults to false)", func(t *testing.T) {
		noLoopYAML := `frames:
  - "frame1.png"
frameSpeed: 0.1
`
		fs := fstest.MapFS{
			"noloop.anim.yaml": {Data: []byte(noLoopYAML)},
		}

		store := newAnimationStore(fs)
		clip, err := store.Load("noloop.anim.yaml")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if clip.Loop {
			t.Errorf("Loop = %v, want false (default)", clip.Loop)
		}
	})

	t.Run("allows atlasPath to be optional (empty string)", func(t *testing.T) {
		fs := fstest.MapFS{
			"anim.anim.yaml": {Data: []byte(validAnimationYAML)},
		}

		store := newAnimationStore(fs)
		clip, err := store.Load("anim.anim.yaml")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if clip.AtlasPath != "" {
			t.Errorf("AtlasPath = %q, want empty string (default)", clip.AtlasPath)
		}
	})

	t.Run("loads multiple clips independently", func(t *testing.T) {
		fs := fstest.MapFS{
			"anim1.anim.yaml": {Data: []byte(validAnimationYAML)},
			"anim2.anim.yaml": {Data: []byte(validAtlasAnimationYAML)},
		}

		store := newAnimationStore(fs)
		clip1, _ := store.Load("anim1.anim.yaml")
		clip2, _ := store.Load("anim2.anim.yaml")

		if clip1.AtlasPath == clip2.AtlasPath {
			t.Errorf("clips should have different atlasPath values")
		}
		if clip1.FrameSpeed == clip2.FrameSpeed {
			t.Errorf("clips should have different frame speeds")
		}
	})
}

func TestAnimationClipStructure(t *testing.T) {
	t.Run("AnimationClip has correct fields", func(t *testing.T) {
		fs := fstest.MapFS{
			"anim.anim.yaml": {Data: []byte(validAnimationYAML)},
		}

		store := newAnimationStore(fs)
		clip, _ := store.Load("anim.anim.yaml")

		if len(clip.Frames) == 0 {
			t.Fatal("expected non-empty Frames")
		}
		if clip.FrameSpeed <= 0 {
			t.Fatal("expected positive FrameSpeed")
		}
		// Loop and AtlasPath are validated by test cases above
	})
}

func TestAnimationStoreThreadSafety(t *testing.T) {
	t.Run("concurrent loads don't cause races", func(t *testing.T) {
		fs := fstest.MapFS{
			"anim.anim.yaml": {Data: []byte(validAnimationYAML)},
		}

		store := newAnimationStore(fs)

		// Simulate concurrent loads
		done := make(chan error, 2)
		go func() {
			_, err := store.Load("anim.anim.yaml")
			done <- err
		}()
		go func() {
			_, err := store.Load("anim.anim.yaml")
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

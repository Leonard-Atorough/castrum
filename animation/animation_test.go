package animation

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/leonard-atorough/castrum/atlas"
)

type mockManager struct {
	storedClip *AnimationClip
	storedID   string
}

func (m *mockManager) store(id string, clip *AnimationClip) {
	m.storedID = id
	m.storedClip = clip
}

func TestAnimationClipBuilder(t *testing.T) {

	t.Run("Gets a new animation clip builder", func(t *testing.T) {
		builder := NewAnimationClipBuilder("test_id", createTextureAtlasForTest(t, "frame1", "frame2"), &mockManager{})
		if builder == nil {
			t.Errorf("expected builder to be non-nil")
		}
	})

	t.Run("Builds animation when provided valid frames and properties", func(t *testing.T) {
		builder := NewAnimationClipBuilder("test_id", createTextureAtlasForTest(t, "frame1", "frame2"), &mockManager{})
		builder.AddFrame("frame1").AddFrame("frame2").SetFrameSpeed(0.5).SetLoop(true)
		clip, err := builder.Build()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if clip == nil {
			t.Errorf("expected clip to be non-nil")
		}
	})

	t.Run("Accepts a list of frames at once", func(t *testing.T) {
		builder := NewAnimationClipBuilder("test_id", createTextureAtlasForTest(t, "frame1", "frame2", "frame3"), &mockManager{})
		builder.AddFrames("frame1", "frame2", "frame3").SetFrameSpeed(0.5).SetLoop(true)
		clip, err := builder.Build()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if clip == nil {
			t.Errorf("expected clip to be non-nil")
		}
	})

	t.Run("Build calls store with correct ID and clip", func(t *testing.T) {
		mockMgr := &mockManager{}
		builder := NewAnimationClipBuilder("test_id", createTextureAtlasForTest(t, "frame1"), mockMgr)
		builder.AddFrame("frame1").SetFrameSpeed(0.5)
		clip, _ := builder.Build()
		if mockMgr.storedID != "test_id" {
			t.Errorf("expected stored ID to be 'test_id', got %q", mockMgr.storedID)
		}
		if mockMgr.storedClip != clip {
			t.Error("expected stored clip to match built clip")
		}
	})

	tests := []struct {
		name        string
		atlas       *atlas.TextureAtlas
		frames      []string
		frameSpeed  float64
		wantErr     bool
		errContains string
	}{
		{"nil atlas", nil, []string{"frame1"}, 0.5, true, "atlas is required"},
		{"no frames", createTextureAtlasForTest(t), []string{}, 0.5, true, "at least one frame"},
		{"zero speed", createTextureAtlasForTest(t, "frame1"), []string{"frame1"}, 0, true, "frame speed must be positive"},
		{"negative speed", createTextureAtlasForTest(t, "frame1"), []string{"frame1"}, -0.5, true, "frame speed must be positive"},
		{"invalid frame", createTextureAtlasForTest(t, "frame1"), []string{"frame2"}, 0.5, true, "region"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAnimationClipBuilder("test", tt.atlas, &mockManager{})
			builder.frames = tt.frames
			builder.frameSpeed = tt.frameSpeed
			_, err := builder.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}

	t.Run("Panics if store is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic when store is nil, got none")
			}
		}()
		_ = NewAnimationClipBuilder("test_id", createTextureAtlasForTest(t, "frame1", "frame2"), nil)
	})
}

func TestAnimationClipStore(t *testing.T) {
	t.Run("Creates an animation clip store", func(t *testing.T) {
		store := NewAnimationClipStore()
		if store == nil {
			t.Errorf("expected a new animation clip store, got nil")
		}
	})

	t.Run("NewClip initializes correctly with correct values", func(t *testing.T) {
		store := NewAnimationClipStore()
		clipBuilder := store.NewClip("test_animation", createTextureAtlasForTest(t))
		if clipBuilder == nil {
			t.Errorf("expected a new animation clip builder, got nil")
		}
		if clipBuilder.id != "test_animation" {
			t.Errorf("expected clip builder ID to be 'test_animation', got %q", clipBuilder.id)
		}
		if clipBuilder.atlas == nil {
			t.Errorf("expected clip builder atlas to be non-nil")
		}
		if clipBuilder.store == nil {
			t.Errorf("expected clip builder manager to be non-nil")
		}
		if len(clipBuilder.frames) != 0 {
			t.Errorf("expected clip builder frames to be empty initially")
		}
		if clipBuilder.frameSpeed != 0 {
			t.Errorf("expected clip builder frame speed to be 0 initially")
		}
		if clipBuilder.loop != false {
			t.Errorf("expected clip builder loop to be false initially")
		}
	})
	// don't use clip builder in this test, directly store and retrieve clips from the store
	t.Run("Stores and gets animation clips from store", func(t *testing.T) {
		store := NewAnimationClipStore()
		var wg sync.WaitGroup
		for i := range 1000 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				clip := &AnimationClip{Frames: []string{fmt.Sprintf("frame%d", i)}}
				store.store(fmt.Sprintf("test_animation_%d", i), clip)
				retrievedClip := store.Get(fmt.Sprintf("test_animation_%d", i))
				if retrievedClip == nil {
					t.Errorf("expected to retrieve the stored clip, got nil")
				}
				if retrievedClip != clip {
					t.Errorf("expected retrieved clip to be the same as the stored clip")
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Overwrites existing clip", func(t *testing.T) {
		store := NewAnimationClipStore()
		clip1 := &AnimationClip{Frames: []string{"frame1"}}
		clip2 := &AnimationClip{Frames: []string{"frame2"}}
		store.store("test", clip1)
		store.store("test", clip2)
		if store.Get("test") != clip2 {
			t.Error("expected clip to be overwritten")
		}
	})

	t.Run("Get returns nil for unknown ID", func(t *testing.T) {
		store := NewAnimationClipStore()
		if store.Get("nonexistent") != nil {
			t.Error("expected nil for unknown ID")
		}
	})
}

func createTextureAtlasForTest(t *testing.T, regionNames ...string) *atlas.TextureAtlas {
	t.Helper()
	regions := make(map[string]*atlas.SubTexture, len(regionNames))
	for _, name := range regionNames {
		regions[name] = &atlas.SubTexture{
			Name:   name,
			Width:  32,
			Height: 32,
		}
	}
	return &atlas.TextureAtlas{
		ID:      "test_atlas",
		TexW:    256,
		TexH:    256,
		Regions: regions,
	}
}

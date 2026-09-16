package animation

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/atlas"
)

func testAtlas() *atlas.Atlas {
	return atlas.NewTextureAtlas(
		"test_atlas",
		"test_texture.png",
		96, 28,
		map[string]atlas.AtlasRegion{
			"frame_0": {Name: "frame_0", X: 0, Y: 0, W: 16, H: 28},
			"frame_1": {Name: "frame_1", X: 16, Y: 0, W: 16, H: 28},
			"frame_2": {Name: "frame_2", X: 32, Y: 0, W: 16, H: 28},
			"frame_3": {Name: "frame_3", X: 48, Y: 0, W: 16, H: 28},
			"frame_4": {Name: "frame_4", X: 64, Y: 0, W: 16, H: 28},
			"frame_5": {Name: "frame_5", X: 80, Y: 0, W: 16, H: 28},
		},
	)
}

// ---------------------------------------------------------------------------
// ClipStore
// ---------------------------------------------------------------------------

func TestClipStoreGetMissingReturnsFalse(t *testing.T) {
	store := NewClipStore()
	clip, ok := store.Get("nonexistent")
	if ok {
		t.Error("Get() for missing clip returned ok=true, want false")
	}
	if clip != nil {
		t.Error("Get() for missing clip returned non-nil clip")
	}
}

func TestClipStoreAddAndGet(t *testing.T) {
	store := NewClipStore()
	clip := &AnimationClip{AtlasID: "a", Frames: []string{"f0"}, FPS: 10}
	store.AddClip("idle", clip)

	got, ok := store.Get("idle")
	if !ok {
		t.Fatal("Get() returned !ok after AddClip")
	}
	if got != clip {
		t.Error("Get() returned a different clip than the one added")
	}
}

func TestClipStoreNewBuilderWiresStore(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	b := store.NewBuilder("walk", a)
	if b == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if b.store != store {
		t.Error("NewBuilder() did not wire the store")
	}
}

// ---------------------------------------------------------------------------
// ClipBuilder.Build
// ---------------------------------------------------------------------------

func TestBuildValidatesNilAtlas(t *testing.T) {
	store := NewClipStore()
	b := &ClipBuilder{id: "test", store: store, fps: 10}
	b.AddFrame("frame_0")
	_, err := b.Build()
	if err == nil {
		t.Fatal("Build() with nil atlas returned nil error")
	}
}

func TestBuildValidatesNoFrames(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	_, err := store.NewBuilder("idle", a).SetFPS(10).Build()
	if err == nil {
		t.Fatal("Build() with no frames returned nil error")
	}
}

func TestBuildValidatesZeroFPS(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	_, err := store.NewBuilder("idle", a).AddFrame("frame_0").Build()
	if err == nil {
		t.Fatal("Build() with zero FPS returned nil error")
	}
}

func TestBuildValidatesNegativeFPS(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	_, err := store.NewBuilder("idle", a).AddFrame("frame_0").SetFPS(-5).Build()
	if err == nil {
		t.Fatal("Build() with negative FPS returned nil error")
	}
}

func TestBuildValidatesUnknownFrame(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	_, err := store.NewBuilder("idle", a).
		AddFrame("frame_0").
		AddFrame("missing_frame").
		SetFPS(10).
		Build()
	if err == nil {
		t.Fatal("Build() with unknown frame returned nil error")
	}
}

func TestBuildCollectsAllErrors(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	_, err := store.NewBuilder("idle", a).
		AddFrame("frame_0").
		AddFrame("missing_a").
		AddFrame("missing_b").
		SetFPS(0).
		Build()
	if err == nil {
		t.Fatal("Build() returned nil error, want collected errors")
	}
	list, ok := err.(*ClipErrorList)
	if !ok {
		t.Fatalf("Build() error type = %T, want *ClipErrorList", err)
	}
	// Expect one error per missing frame plus one for FPS.
	if len(list.Errors) != 3 {
		t.Fatalf("ClipErrorList has %d errors, want 3: %v", len(list.Errors), list.Errors)
	}
}

func TestBuildRegistersClip(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	clip, err := store.NewBuilder("walk", a).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFPS(12).
		SetLoop(LoopForever).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	got, ok := store.Get("walk")
	if !ok {
		t.Fatal("clip not registered in store after Build()")
	}
	if got != clip {
		t.Error("registered clip differs from returned clip")
	}
}

func TestBuildStoresAtlasID(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	clip, err := store.NewBuilder("idle", a).
		AddFrame("frame_0").
		SetFPS(10).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if clip.AtlasID != a.ID() {
		t.Errorf("AtlasID = %q, want %q", clip.AtlasID, a.ID())
	}
}

// ---------------------------------------------------------------------------
// ClipBuilder defensive copies
// ---------------------------------------------------------------------------

func TestBuildCopiesFrames(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	b := store.NewBuilder("idle", a)
	b.AddFrame("frame_0").AddFrame("frame_1").SetFPS(10)
	clip, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Mutate the builder after Build — the clip must not be affected.
	b.AddFrame("frame_2")

	if len(clip.Frames) != 2 {
		t.Errorf("clip.Frames has %d frames after builder mutation, want 2", len(clip.Frames))
	}
}

func TestBuildCopiesFrameEvents(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	b := store.NewBuilder("idle", a)
	b.AddFrame("frame_0").AddFrame("frame_1").SetFPS(10)
	b.OnFrame(1, "hit")
	clip, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Mutate the builder after Build.
	b.OnFrame(0, "start")

	if names := clip.FrameEvents[0]; len(names) != 0 {
		t.Errorf("clip.FrameEvents[0] has %d names after builder mutation, want 0", len(names))
	}
	names := clip.FrameEvents[1]
	if len(names) != 1 || names[0] != "hit" {
		t.Errorf("clip.FrameEvents[1] = %v, want [hit]", names)
	}
}

// ---------------------------------------------------------------------------
// ClipBuilder fluent methods
// ---------------------------------------------------------------------------

func TestAddFramesAppendsAll(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	clip, err := store.NewBuilder("idle", a).
		AddFrames("frame_0", "frame_1", "frame_2").
		SetFPS(10).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(clip.Frames) != 3 {
		t.Errorf("clip.Frames has %d frames, want 3", len(clip.Frames))
	}
}

func TestOnFrameAccumulatesMultipleEvents(t *testing.T) {
	store := NewClipStore()
	a := testAtlas()
	b := store.NewBuilder("idle", a)
	b.AddFrame("frame_0").SetFPS(10)
	b.OnFrame(0, "hit").OnFrame(0, "effect")
	clip, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	names := clip.FrameEvents[0]
	if len(names) != 2 {
		t.Fatalf("FrameEvents[0] has %d names, want 2", len(names))
	}
	if names[0] != "hit" || names[1] != "effect" {
		t.Errorf("FrameEvents[0] = %v, want [hit effect]", names)
	}
}

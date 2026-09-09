package animation

import (
	"image/color"
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/events"
)

func setupTestWorld() *core.World {
	world := core.NewWorld()
	core.SetResource(world, events.NewEventBus())
	return world
}

func createTestAtlas() *assets.TextureAtlas {
	// Create a minimal test atlas with a fake ebiten.Image
	return &assets.TextureAtlas{
		Path: "test.json",
		Regions: map[string]*assets.SubTexture{
			"frame_0": {Name: "frame_0", Width: 32, Height: 32},
			"frame_1": {Name: "frame_1", Width: 32, Height: 32},
			"frame_2": {Name: "frame_2", Width: 32, Height: 32},
		},
	}
}

func createAnimatingEntity(world *core.World, clipID string) core.EntityID {
	entity, _ := world.CreateWithComponents("test_entity",
		components.Transform{
			Position: geom.Vector2{X: 0, Y: 0},
			Scale:    geom.Vector2{X: 1, Y: 1},
			Color:    color.White,
		},
		components.Animation{
			ClipPath:      clipID,
			FrameIndex:    0,
			FrameTime:     0,
			Playing:       true,
			PlaybackSpeed: 1.0,
		},
		components.Sprite{
			Visible: true,
		},
	)
	return entity.ID
}

func TestSystem_Init(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	err := sys.Init(world)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestSystem_Update_AdvancesFrameTime(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	// Create a test clip
	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFrameSpeed(0.1).
		SetLoop(false).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")

	err = sys.Update(world, 0.05)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameTime < 0.04 {
		t.Errorf("FrameTime should advance, got %v", anim.FrameTime)
	}
	if anim.FrameIndex != 0 {
		t.Error("FrameIndex should still be 0 (threshold not met)")
	}
}

func TestSystem_Update_AdvancesFrame(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFrameSpeed(0.1).
		SetLoop(false).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")

	err = sys.Update(world, 0.15)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1", anim.FrameIndex)
	}
}

func TestSystem_Update_EmitsLoopEvent(t *testing.T) {
	world := setupTestWorld()
	bus, _ := core.GetResource[*events.EventBus](world)
	var emittedEvent AnimationEvent
	var eventFired bool

	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		emittedEvent = e
		eventFired = true
	}, false)

	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFrameSpeed(0.1).
		SetLoop(true).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.FrameIndex = 1
	world.SetComponent(entity, anim)

	sys.Update(world, 0.15)

	if !eventFired {
		t.Fatal("expected loop event to be emitted")
	}
	if emittedEvent.Type != EventClipLooped {
		t.Errorf("Event type = %d, want EventClipLooped (%d)", emittedEvent.Type, EventClipLooped)
	}
}

func TestSystem_Update_IgnoresNonPlayingAnimations(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFrameSpeed(0.1).
		SetLoop(false).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.Playing = false
	world.SetComponent(entity, anim)

	sys.Update(world, 1.0)

	anim, _ = world.GetComponent[components.Animation](entity)
	if anim.FrameTime != 0 {
		t.Error("FrameTime should not change for non-playing animation")
	}
}

func TestSystem_Update_LoopsAnimation(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFrameSpeed(0.1).
		SetLoop(true).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.FrameIndex = 1
	world.SetComponent(entity, anim)

	sys.Update(world, 0.15)

	anim, _ = world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 after loop", anim.FrameIndex)
	}
	if !anim.Playing {
		t.Error("Animation should still be playing after loop")
	}
}

func TestSystem_Update_StopsNonLoopingAnimation(t *testing.T) {
	world := setupTestWorld()
	bus, _ := core.GetResource[*events.EventBus](world)
	var emittedEvent AnimationEvent
	var eventFired bool

	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		emittedEvent = e
		eventFired = true
	}, false)

	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		SetFrameSpeed(0.1).
		SetLoop(false).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.FrameIndex = 1
	world.SetComponent(entity, anim)

	sys.Update(world, 0.15)

	if !eventFired {
		t.Fatal("expected completion event to be emitted")
	}
	if emittedEvent.Type != EventClipFinished {
		t.Errorf("Event type = %d, want EventClipFinished (%d)", emittedEvent.Type, EventClipFinished)
	}

	anim, _ = world.GetComponent[components.Animation](entity)
	if anim.Playing {
		t.Error("Animation should stop after reaching end")
	}
}

func TestSystem_Update_RespectPlaybackSpeed(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	atlas := createTestAtlas()
	_, err := mgr.NewClip("test_clip", atlas).
		AddFrame("frame_0").
		AddFrame("frame_1").
		AddFrame("frame_2").
		SetFrameSpeed(0.1).
		SetLoop(false).
		Build()
	if err != nil {
		t.Fatalf("Failed to build clip: %v", err)
	}

	entity := createAnimatingEntity(world, "test_clip")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.PlaybackSpeed = 2.0 // double speed
	world.SetComponent(entity, anim)

	// At 2x speed, 0.05 delta should advance frame time by 0.1 (threshold for 0.1 frame speed)
	sys.Update(world, 0.05)

	anim, _ = world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 1 {
		t.Errorf("At 2x speed, FrameIndex = %d, want 1", anim.FrameIndex)
	}
}

func TestSystem_Update_SkipsMissingClips(t *testing.T) {
	world := setupTestWorld()
	mgr := NewManager()
	sys := NewSystem(mgr)
	sys.Init(world)

	// Don't add clip to manager
	entity := createAnimatingEntity(world, "missing_clip")

	err := sys.Update(world, 0.1)
	if err != nil {
		t.Fatalf("Update should not fail on missing clip: %v", err)
	}

	// Animation should remain unchanged
	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 0 {
		t.Error("Frame index should not change when clip is missing")
	}
}

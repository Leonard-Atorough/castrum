package animation

import (
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

func createAnimatingEntity(world *core.World, clipPath string) core.EntityID {
	entity, _ := world.CreateWithComponents("test_entity",
		components.Transform{
			Position: geom.Vector2{X: 0, Y: 0},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		components.Animation{
			ClipPath:      clipPath,
			FrameIndex:    0,
			FrameTime:     0,
			Playing:       true,
			PlaybackSpeed: 1.0,
		},
		components.Renderable{
			Visible: true,
		},
	)
	return entity.ID
}

func TestSystem_Init(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	err := sys.Init(world)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestSystem_Shutdown(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)
	err := sys.Shutdown(world)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestSystem_Update_AdvancesFrameTime(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	// Setup mock clip
	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Loop:       false,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")

	err := sys.Update(world, 0.05)
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
	sys := &System{}
	sys.Init(world)

	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Loop:       false,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")

	err := sys.Update(world, 0.15)
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
	sys := &System{}
	sys.Init(world)

	bus, _ := core.GetResource[*events.EventBus](world)
	var emittedEvent AnimationEvent
	var eventFired bool

	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		emittedEvent = e
		eventFired = true
	}, false)

	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Loop:       true,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")
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
	sys := &System{}
	sys.Init(world)

	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Loop:       false,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")
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
	sys := &System{}
	sys.Init(world)

	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Loop:       true,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")
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
	sys := &System{}
	sys.Init(world)

	bus, _ := core.GetResource[*events.EventBus](world)
	var emittedEvent AnimationEvent
	var eventFired bool

	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		emittedEvent = e
		eventFired = true
	}, false)

	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Loop:       false,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")
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
	sys := &System{}
	sys.Init(world)

	clip := &assets.AnimationClip{
		Frames:     []string{"frame_0.png", "frame_1.png", "frame_2.png"},
		FrameSpeed: 0.1,
		Loop:       false,
	}
	sys.assetLoader.Animations.Animations["test.anim.yaml"] = clip

	entity := createAnimatingEntity(world, "test.anim.yaml")
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
	sys := &System{}
	sys.Init(world)

	// Don't add clip to asset loader
	entity := createAnimatingEntity(world, "missing.anim.yaml")

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

func TestComponentControl_Play(t *testing.T) {
	world := setupTestWorld()
	entity := createAnimatingEntity(world, "test.anim.yaml")

	// Direct component modification to play
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.Playing = true
	anim.FrameTime = 0
	anim.FrameIndex = 0
	world.SetComponent(entity, anim)

	// Verify
	anim, _ = world.GetComponent[components.Animation](entity)
	if !anim.Playing {
		t.Error("Animation should be playing after modification")
	}
	if anim.FrameIndex != 0 {
		t.Error("FrameIndex should be 0")
	}
}

func TestComponentControl_Pause(t *testing.T) {
	world := setupTestWorld()
	entity := createAnimatingEntity(world, "test.anim.yaml")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.FrameIndex = 1
	world.SetComponent(entity, anim)

	// Direct component modification to pause
	anim, _ = world.GetComponent[components.Animation](entity)
	anim.Playing = false
	world.SetComponent(entity, anim)

	// Verify
	anim, _ = world.GetComponent[components.Animation](entity)
	if anim.Playing {
		t.Error("Animation should not be playing after pause")
	}
	if anim.FrameIndex != 1 {
		t.Error("FrameIndex should remain at 1 after pause")
	}
}

func TestComponentControl_Stop(t *testing.T) {
	world := setupTestWorld()
	entity := createAnimatingEntity(world, "test.anim.yaml")
	anim, _ := world.GetComponent[components.Animation](entity)
	anim.FrameIndex = 1
	anim.FrameTime = 0.05
	world.SetComponent(entity, anim)

	// Direct component modification to stop
	anim, _ = world.GetComponent[components.Animation](entity)
	anim.Playing = false
	anim.FrameIndex = 0
	anim.FrameTime = 0
	world.SetComponent(entity, anim)

	// Verify
	anim, _ = world.GetComponent[components.Animation](entity)
	if anim.Playing {
		t.Error("Animation should not be playing")
	}
	if anim.FrameIndex != 0 {
		t.Error("FrameIndex should be reset to 0")
	}
	if anim.FrameTime != 0 {
		t.Error("FrameTime should be reset to 0")
	}
}

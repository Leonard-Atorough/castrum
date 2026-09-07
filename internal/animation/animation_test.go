package animation

import (
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

func setupTestWorld() *core.World {
	return core.NewWorld()
}

func createAnimatableEntity(world *core.World, animations map[string]components.Animation) core.EntityID {
	entity, _ := world.CreateWithComponents("test_entity",
		components.Transform{
			Position: geom.Vector2{X: 0, Y: 0},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		components.Animatable{
			Animations:       animations,
			CurrentAnimation: "default",
		},
		components.Renderable{
			Visible: true,
		},
	)
	return entity.ID
}

func TestNewSystem(t *testing.T) {
	sys := &System{}
	if sys.Events() == nil {
		t.Error("Events should return slice, not nil")
	}
	if len(sys.Events()) != 0 {
		t.Error("Initial events should be empty")
	}
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

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameTime:  0,
		FrameIndex: 0,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	err := sys.Update(world, 0.05)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	animComp, _ := world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].FrameTime < 0.04 {
		t.Errorf("FrameTime should advance, got %v", animComp.Animations["default"].FrameTime)
	}
	if animComp.Animations["default"].FrameIndex != 0 {
		t.Error("FrameIndex should still be 0 (threshold not met)")
	}
}

func TestSystem_Update_AdvancesFrame(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameTime:  0.05,
		FrameIndex: 0,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	err := sys.Update(world, 0.1)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	animComp, _ := world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1", animComp.Animations["default"].FrameIndex)
	}
}

func TestSystem_Update_EmitsFrameEvent(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameTime:  0.05,
		FrameIndex: 0,
	}
	_ = createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	sys.Update(world, 0.1)

	events := sys.Events()
	if len(events) == 0 {
		t.Fatal("expected frame event to be emitted")
	}
	if events[0].Type != FrameEventType {
		t.Errorf("Event type = %d, want FrameEventType", events[0].Type)
	}
	if events[0].FrameIndex != 1 {
		t.Errorf("Event FrameIndex = %d, want 1", events[0].FrameIndex)
	}
}

func TestSystem_Update_IgnoresNonPlayingAnimations(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    false,
		FrameTime:  0,
		FrameIndex: 0,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	sys.Update(world, 1.0)

	animComp, _ := world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].FrameTime != 0 {
		t.Error("FrameTime should not change for non-playing animation")
	}
}

func TestSystem_Update_LoopsAnimation(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameTime:  0.05,
		FrameIndex: 1,
		Loop:       true,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	sys.Update(world, 0.1)

	animComp, _ := world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 after loop", animComp.Animations["default"].FrameIndex)
	}
	if !animComp.Animations["default"].Playing {
		t.Error("Animation should still be playing after loop")
	}
}

func TestSystem_Update_StopsNonLoopingAnimation(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameTime:  0.05,
		FrameIndex: 1,
		Loop:       false,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	sys.Update(world, 0.1)

	animComp, _ := world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].Playing {
		t.Error("Animation should stop when reaching end with Loop=false")
	}
	if animComp.Animations["default"].FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1", animComp.Animations["default"].FrameIndex)
	}
}

func TestSystem_Update_EmitsCompleteEvent(t *testing.T) {
	world := setupTestWorld()
	sys := &System{}
	sys.Init(world)

	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameTime:  0.05,
		FrameIndex: 1,
		Loop:       false,
	}
	_ = createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	sys.Update(world, 0.1)

	events := sys.Events()
	hasComplete := false
	for _, e := range events {
		if e.Type == CompleteEventType {
			hasComplete = true
			break
		}
	}
	if !hasComplete {
		t.Fatal("expected complete event to be emitted")
	}
}

func TestComponentControl_Play(t *testing.T) {
	world := setupTestWorld()
	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    false,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	// Direct component modification to play
	animComp, _ := world.GetComponent[components.Animatable](entity)
	current := animComp.Animations["default"]
	current.Playing = true
	current.FrameTime = 0
	current.FrameIndex = 0
	animComp.Animations["default"] = current
	world.SetComponent(entity, animComp)

	// Verify
	animComp, _ = world.GetComponent[components.Animatable](entity)
	if !animComp.Animations["default"].Playing {
		t.Error("Animation should be playing after modification")
	}
	if animComp.Animations["default"].FrameIndex != 0 {
		t.Error("FrameIndex should be 0")
	}
}

func TestComponentControl_Pause(t *testing.T) {
	world := setupTestWorld()
	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameIndex: 1,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	// Direct component modification to pause
	animComp, _ := world.GetComponent[components.Animatable](entity)
	current := animComp.Animations["default"]
	current.Playing = false
	animComp.Animations["default"] = current
	world.SetComponent(entity, animComp)

	// Verify
	animComp, _ = world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].Playing {
		t.Error("Animation should not be playing after pause")
	}
	if animComp.Animations["default"].FrameIndex != 1 {
		t.Error("FrameIndex should remain at 1 after pause")
	}
}

func TestComponentControl_Stop(t *testing.T) {
	world := setupTestWorld()
	anim := components.Animation{
		Frames:     []string{"frame_0.png", "frame_1.png"},
		FrameSpeed: 0.1,
		Playing:    true,
		FrameIndex: 1,
		FrameTime:  0.05,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"default": anim,
	})

	// Direct component modification to stop
	animComp, _ := world.GetComponent[components.Animatable](entity)
	current := animComp.Animations["default"]
	current.Playing = false
	current.FrameIndex = 0
	current.FrameTime = 0
	animComp.Animations["default"] = current
	world.SetComponent(entity, animComp)

	// Verify
	animComp, _ = world.GetComponent[components.Animatable](entity)
	if animComp.Animations["default"].Playing {
		t.Error("Animation should not be playing")
	}
	if animComp.Animations["default"].FrameIndex != 0 {
		t.Error("FrameIndex should be reset to 0")
	}
	if animComp.Animations["default"].FrameTime != 0 {
		t.Error("FrameTime should be reset to 0")
	}
}

func TestComponentControl_SwitchAnimation(t *testing.T) {
	world := setupTestWorld()
	anim1 := components.Animation{
		Frames:     []string{"a1.png", "a2.png"},
		FrameSpeed: 0.1,
	}
	anim2 := components.Animation{
		Frames:     []string{"b1.png", "b2.png"},
		FrameSpeed: 0.2,
	}
	entity := createAnimatableEntity(world, map[string]components.Animation{
		"anim1": anim1,
		"anim2": anim2,
	})

	// Direct component modification to switch animation
	animComp, _ := world.GetComponent[components.Animatable](entity)
	animComp.CurrentAnimation = "anim2"
	world.SetComponent(entity, animComp)

	// Verify
	animComp, _ = world.GetComponent[components.Animatable](entity)
	if animComp.CurrentAnimation != "anim2" {
		t.Errorf("CurrentAnimation = %s, want anim2", animComp.CurrentAnimation)
	}
}

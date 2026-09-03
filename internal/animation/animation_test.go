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
	entity, _ := world.CreateWithComponents("",
		components.Transform{
			Position: geom.Vector2{X: 0, Y: 0},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		components.Animatable{
			Animations:       animations,
			CurrentAnimation: "default",
		},
	)
	return entity.ID
}

func TestManager(t *testing.T) {
	t.Run("lifecycle", func(t *testing.T) {
		t.Run("NewManager initializes with empty events", func(t *testing.T) {
			am := NewManager()
			if am == nil {
				t.Fatal("NewManager returned nil")
			}
			if am.Events() == nil {
				t.Error("Events should return slice, not nil")
			}
			if len(am.Events()) != 0 {
				t.Error("Initial events should be empty")
			}
		})
	})

	t.Run("controls", func(t *testing.T) {
		t.Run("Play starts animation from frame 0", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()
			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    false,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			err := am.Play(world, entity)
			if err != nil {
				t.Fatalf("Play failed: %v", err)
			}

			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			if !animComp.Animations["default"].Playing {
				t.Error("Animation should be playing after Play()")
			}
			if animComp.Animations["default"].FrameIndex != 0 {
				t.Error("FrameIndex should reset to 0")
			}
			if animComp.Animations["default"].FrameTime != 0 {
				t.Error("FrameTime should reset to 0")
			}
		})

		t.Run("Play fails on nonexistent entity", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()
			err := am.Play(world, 9999)
			if err == nil {
				t.Error("Play should fail for nonexistent entity")
			}
		})

		t.Run("Pause stops animation without reset", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()
			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			err := am.Pause(world, entity)
			if err != nil {
				t.Fatalf("Pause failed: %v", err)
			}

			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			if animComp.Animations["default"].Playing {
				t.Error("Animation should not be playing after Pause()")
			}
		})

		t.Run("Stop halts and resets animation", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()
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

			err := am.Stop(world, entity)
			if err != nil {
				t.Fatalf("Stop failed: %v", err)
			}

			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			current := animComp.Animations["default"]
			if current.Playing {
				t.Error("Animation should not be playing after Stop()")
			}
			if current.FrameIndex != 0 {
				t.Error("FrameIndex should reset to 0")
			}
			if current.FrameTime != 0 {
				t.Error("FrameTime should reset to 0")
			}
		})

		t.Run("Reset clears state without playing", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()
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

			err := am.Reset(world, entity)
			if err != nil {
				t.Fatalf("Reset failed: %v", err)
			}

			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			current := animComp.Animations["default"]
			if current.Playing {
				t.Error("Animation should not be playing after Reset()")
			}
			if current.FrameIndex != 0 {
				t.Error("FrameIndex should be 0")
			}
			if current.FrameTime != 0 {
				t.Error("FrameTime should be 0")
			}
		})
	})

	t.Run("switching", func(t *testing.T) {
		t.Run("SwitchAnimation changes current animation", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim1 := components.Animation{
				Frames:     []string{"walk_0.png", "walk_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
			}
			anim2 := components.Animation{
				Frames:     []string{"jump_0.png", "jump_1.png"},
				FrameSpeed: 0.05,
				Playing:    false,
			}

			entity := createAnimatableEntity(world, map[string]components.Animation{
				"walk": anim1,
				"jump": anim2,
			})

			err := am.SwitchAnimation(world, entity, "jump")
			if err != nil {
				t.Fatalf("SwitchAnimation failed: %v", err)
			}

			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			if animComp.CurrentAnimation != "jump" {
				t.Errorf("CurrentAnimation should be 'jump', got %s", animComp.CurrentAnimation)
			}
		})

		t.Run("SwitchAnimation fails on nonexistent animation", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png"},
				FrameSpeed: 0.1,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			err := am.SwitchAnimation(world, entity, "nonexistent")
			if err == nil {
				t.Error("SwitchAnimation should fail for nonexistent animation")
			}
		})
	})

	t.Run("frame advancement", func(t *testing.T) {
		t.Run("Update advances frame when delta exceeds FrameSpeed", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png", "frame_2.png"},
				FrameSpeed: 0.1,
				Playing:    true,
				FrameIndex: 0,
				FrameTime:  0,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			// Insufficient delta
			am.Update(world, 0.05)
			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			if animComp.Animations["default"].FrameIndex != 0 {
				t.Error("Frame should not advance with delta < FrameSpeed")
			}

			// Sufficient delta
			am.Update(world, 0.1)
			animComp, _ = core.GetComponent[components.Animatable](world, entity)
			if animComp.Animations["default"].FrameIndex != 1 {
				t.Errorf("Frame should advance to 1, got %d", animComp.Animations["default"].FrameIndex)
			}
		})

		t.Run("Looping animation cycles frames", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Loop:       true,
				Playing:    true,
				FrameIndex: 1,
				FrameTime:  0,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			if animComp.Animations["default"].FrameIndex != 0 {
				t.Errorf("Looping animation should reset to 0, got %d", animComp.Animations["default"].FrameIndex)
			}
			if !animComp.Animations["default"].Playing {
				t.Error("Looping animation should still be playing")
			}
		})

		t.Run("Non-looping animation stops at last frame", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Loop:       false,
				Playing:    true,
				FrameIndex: 1,
				FrameTime:  0,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			if animComp.Animations["default"].FrameIndex != 1 {
				t.Errorf("Non-looping animation should stay at last frame, got %d", animComp.Animations["default"].FrameIndex)
			}
			if animComp.Animations["default"].Playing {
				t.Error("Non-looping animation should stop")
			}
		})
	})

	t.Run("events", func(t *testing.T) {
		t.Run("Update emits FrameEventType when frame advances", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
				FrameIndex: 0,
				FrameTime:  0,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			events := am.Events()

			if len(events) == 0 {
				t.Fatal("Expected frame event")
			}

			frameEvent := events[0]
			if frameEvent.Type != FrameEventType {
				t.Errorf("Expected FrameEventType, got %d", frameEvent.Type)
			}
			if frameEvent.EntityID != entity {
				t.Errorf("Event EntityID mismatch")
			}
			if frameEvent.FrameIndex != 1 {
				t.Errorf("Expected frame index 1, got %d", frameEvent.FrameIndex)
			}
		})

		t.Run("Update emits CompleteEventType when animation finishes", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Loop:       false,
				Playing:    true,
				FrameIndex: 1,
				FrameTime:  0,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			events := am.Events()

			completeFound := false
			for _, e := range events {
				if e.Type == CompleteEventType && e.EntityID == entity {
					completeFound = true
					break
				}
			}

			if !completeFound {
				t.Fatal("Expected completion event")
			}
		})

		t.Run("Events clears after each Update", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
				FrameIndex: 0,
				FrameTime:  0,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			// First update
			am.Update(world, 0.1)
			if len(am.Events()) == 0 {
				t.Fatal("First update should have events")
			}

			// Second update with stopped animation
			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			current := animComp.Animations["default"]
			current.Playing = false
			animComp.Animations["default"] = current
			core.SetComponent(world, entity, animComp)

			am.Update(world, 0.1)
			if len(am.Events()) != 0 {
				t.Errorf("Events should be cleared for next frame, got %d", len(am.Events()))
			}
		})

		t.Run("Multiple entities emit multiple events", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
				FrameIndex: 0,
				FrameTime:  0,
			}
			createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})
			createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			events := am.Events()

			if len(events) != 2 {
				t.Errorf("Expected 2 events, got %d", len(events))
			}
		})
	})

	t.Run("callbacks", func(t *testing.T) {
		t.Run("Frame callback fires when frame changes", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			callbackCalled := false
			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
				FrameIndex: 0,
				FrameTime:  0,
				FrameEvents: map[int]func(){
					1: func() {
						callbackCalled = true
					},
				},
			}
			createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			if !callbackCalled {
				t.Error("Frame callback should have been called")
			}
		})

		t.Run("Completion callback fires when animation ends", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			callbackCalled := false
			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Loop:       false,
				Playing:    true,
				FrameIndex: 1,
				FrameTime:  0,
				Callback: func() {
					callbackCalled = true
				},
			}
			createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			am.Update(world, 0.1)
			if !callbackCalled {
				t.Error("Completion callback should have been called")
			}
		})
	})

	t.Run("getters", func(t *testing.T) {
		t.Run("IsPlaying returns animation state", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png"},
				FrameSpeed: 0.1,
				Playing:    true,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			playing, err := am.IsPlaying(world, entity)
			if err != nil {
				t.Fatalf("IsPlaying failed: %v", err)
			}
			if !playing {
				t.Error("IsPlaying should return true")
			}
		})

		t.Run("GetFrameIndex returns current frame", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				FrameIndex: 1,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			idx, err := am.GetFrameIndex(world, entity)
			if err != nil {
				t.Fatalf("GetFrameIndex failed: %v", err)
			}
			if idx != 1 {
				t.Errorf("Expected frame index 1, got %d", idx)
			}
		})

		t.Run("GetFrameTime returns elapsed time", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png"},
				FrameSpeed: 0.1,
				FrameTime:  0.05,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			ft, err := am.GetFrameTime(world, entity)
			if err != nil {
				t.Fatalf("GetFrameTime failed: %v", err)
			}
			if ft != 0.05 {
				t.Errorf("Expected frame time 0.05, got %f", ft)
			}
		})

		t.Run("GetFrameSpeed returns frame speed", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png"},
				FrameSpeed: 0.15,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			fs, err := am.GetFrameSpeed(world, entity)
			if err != nil {
				t.Fatalf("GetFrameSpeed failed: %v", err)
			}
			if fs != 0.15 {
				t.Errorf("Expected frame speed 0.15, got %f", fs)
			}
		})

		t.Run("GetFrames returns animation frame lists", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim1 := components.Animation{
				Frames:     []string{"walk_0.png", "walk_1.png"},
				FrameSpeed: 0.1,
			}
			anim2 := components.Animation{
				Frames:     []string{"run_0.png", "run_1.png", "run_2.png"},
				FrameSpeed: 0.05,
			}

			entity := createAnimatableEntity(world, map[string]components.Animation{
				"walk": anim1,
				"run":  anim2,
			})

			frames, err := am.GetFrames(world, entity, "walk", "run")
			if err != nil {
				t.Fatalf("GetFrames failed: %v", err)
			}

			if len(frames["walk"]) != 2 {
				t.Errorf("Expected 2 walk frames, got %d", len(frames["walk"]))
			}
			if len(frames["run"]) != 3 {
				t.Errorf("Expected 3 run frames, got %d", len(frames["run"]))
			}
			if frames["walk"][0] != "walk_0.png" {
				t.Errorf("Expected walk_0.png, got %s", frames["walk"][0])
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("Update skips entity with invalid current animation", func(t *testing.T) {
			world := setupTestWorld()
			am := NewManager()

			anim := components.Animation{
				Frames:     []string{"frame_0.png", "frame_1.png"},
				FrameSpeed: 0.1,
				Playing:    true,
			}
			entity := createAnimatableEntity(world, map[string]components.Animation{
				"default": anim,
			})

			animComp, _ := core.GetComponent[components.Animatable](world, entity)
			animComp.CurrentAnimation = "nonexistent"
			core.SetComponent(world, entity, animComp)

			err := am.Update(world, 0.1)
			if err != nil {
				t.Fatalf("Update should not error on invalid animation: %v", err)
			}
		})
	})
}

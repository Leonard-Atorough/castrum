package castrum

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/geom"
)

func TestNewGame(t *testing.T) {
	t.Run("wires fixedDelta, camera, and shared resources from config", func(t *testing.T) {
		config := DefaultConfig()
		config.Graphics.VirtualWidth = 320
		config.Graphics.VirtualHeight = 240
		config.Engine.TicksPerSecond = 30

		game, err := NewGame(config, nil)
		if err != nil {
			t.Fatalf("NewGame failed: %v", err)
		}

		if want := 1.0 / 30.0; game.fixedDelta != want {
			t.Fatalf("fixedDelta = %v, want %v", game.fixedDelta, want)
		}

		// Check camera entity
		cam, err := game.World.GetComponent[Camera](game.CameraEntityID)
		if err != nil {
			t.Fatalf("Failed to get camera component: %v", err)
		}
		if cam.ScreenSize.X != 320 || cam.ScreenSize.Y != 240 {
			t.Fatalf("Camera.ScreenSize = %v, want {320 240}", cam.ScreenSize)
		}
		if game.renderer == nil || game.World == nil || game.Systems == nil {
			t.Fatal("expected NewGame to wire all core subsystems")
		}
	})

	t.Run("validates a sparse config instead of dividing by zero", func(t *testing.T) {
		game, err := NewGame(&Config{}, nil)
		if err != nil {
			t.Fatalf("NewGame failed: %v", err)
		}

		if game.Config.Engine.TicksPerSecond <= 0 {
			t.Fatalf("expected ValidateConfig to fill in TicksPerSecond, got %d", game.Config.Engine.TicksPerSecond)
		}
		if game.fixedDelta <= 0 {
			t.Fatalf("fixedDelta = %v, want a positive value", game.fixedDelta)
		}
	})
}

func TestGame_Layout(t *testing.T) {
	config := DefaultConfig()
	config.Graphics.VirtualWidth = 640
	config.Graphics.VirtualHeight = 480
	game, err := NewGame(config, nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	w, h := game.Layout(1920, 1080)

	if w != 640 || h != 480 {
		t.Fatalf("Layout() = (%d, %d), want (640, 480)", w, h)
	}

	// Check camera entity after layout
	cam, err := game.World.GetComponent[Camera](game.CameraEntityID)
	if err != nil {
		t.Fatalf("Failed to get camera component: %v", err)
	}
	if cam.ScreenSize.X != 640 || cam.ScreenSize.Y != 480 {
		t.Fatalf("Camera.ScreenSize = %v, want {640 480}", cam.ScreenSize)
	}
}

// TestGame_Update_DoesNotHang is a regression test for a fixedDelta==0 bug:
// Update()'s accumulator loop divides/subtracts by fixedDelta, so an
// uninitialized fixedDelta caused an infinite loop the first time Update ran.
func TestGame_Update_DoesNotHang(t *testing.T) {
	game, err := NewGame(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- game.Update()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Update() returned an error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Update() did not return - possible fixedDelta accumulator hang")
	}
}

func TestGame_Draw_DoesNotPanic(t *testing.T) {
	game, _ := NewGame(DefaultConfig(), nil)
	screen := ebiten.NewImage(game.Config.Graphics.VirtualWidth, game.Config.Graphics.VirtualHeight)

	game.Draw(screen)
}

func TestGame_Camera_Methods(t *testing.T) {
	game, err := NewGame(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	t.Run("GetCamera returns the camera component", func(t *testing.T) {
		cam, err := game.GetCamera()
		if err != nil {
			t.Fatalf("GetCamera failed: %v", err)
		}
		if cam.ScreenSize.X <= 0 || cam.ScreenSize.Y <= 0 {
			t.Fatalf("Camera.ScreenSize should be positive, got %v", cam.ScreenSize)
		}
	})

	t.Run("SetCamera updates the camera component", func(t *testing.T) {
		newCam := Camera{Position: geom.Vector2{X: 100, Y: 200}, Zoom: 2.0}
		err := game.SetCamera(newCam)
		if err != nil {
			t.Fatalf("SetCamera failed: %v", err)
		}

		cam, err := game.GetCamera()
		if err != nil {
			t.Fatalf("GetCamera failed after SetCamera: %v", err)
		}
		if cam.Position.X != 100 || cam.Position.Y != 200 {
			t.Fatalf("SetCamera didn't update position: got %v, want {100 200}", cam.Position)
		}
		if cam.Zoom != 2.0 {
			t.Fatalf("SetCamera didn't update zoom: got %v, want 2.0", cam.Zoom)
		}
	})

	t.Run("GetCameraViewport returns a valid rect", func(t *testing.T) {
		viewport, err := game.GetCameraViewport()
		if err != nil {
			t.Fatalf("GetCameraViewport failed: %v", err)
		}
		// Just verify that we got a rect back without error
		// The bounds depend on camera state initialization
		if (viewport == geom.Rect{}) {
			t.Fatalf("GetCameraViewport returned empty rect")
		}
	})
}

func TestGame_Pause_and_TimeScale(t *testing.T) {
	game, err := NewGame(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	t.Run("SetPaused and IsPaused control pause state", func(t *testing.T) {
		if game.IsPaused() {
			t.Fatal("Game should not be paused initially")
		}

		game.SetPaused(true)
		if !game.IsPaused() {
			t.Fatal("SetPaused(true) didn't set pause state")
		}

		game.SetPaused(false)
		if game.IsPaused() {
			t.Fatal("SetPaused(false) didn't unset pause state")
		}
	})

	t.Run("SetTimeScale and GetTimeScale control speed", func(t *testing.T) {
		initialSpeed := game.GetTimeScale()
		if initialSpeed <= 0 {
			t.Fatalf("Initial time scale should be positive, got %v", initialSpeed)
		}

		game.SetTimeScale(0.5)
		if speed := game.GetTimeScale(); speed != 0.5 {
			t.Fatalf("SetTimeScale(0.5) didn't work: got %v", speed)
		}

		game.SetTimeScale(2.0)
		if speed := game.GetTimeScale(); speed != 2.0 {
			t.Fatalf("SetTimeScale(2.0) didn't work: got %v", speed)
		}

		// Test clamping to zero
		game.SetTimeScale(-1)
		if speed := game.GetTimeScale(); speed != 0 {
			t.Fatalf("SetTimeScale should clamp negative values to 0, got %v", speed)
		}
	})
}

func TestGame_Scenes_Method(t *testing.T) {
	game, err := NewGame(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	scenes := game.Scenes()
	if scenes == nil {
		t.Fatal("Scenes() returned nil")
	}
}

func TestGame_RegisterSystem(t *testing.T) {
	game, err := NewGame(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	mockSystem := &MockSystem{}
	err = game.RegisterSystem("test_system", 0, mockSystem)
	if err != nil {
		t.Fatalf("RegisterSystem failed: %v", err)
	}
}

func TestGame_FindAll_FindOne_CountWith(t *testing.T) {
	game, err := NewGame(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("NewGame failed: %v", err)
	}

	t.Run("CountWith returns count of entities with component type", func(t *testing.T) {
		count := CountWith[Transform](game.World)
		if count < 0 {
			t.Fatalf("CountWith returned negative count: %d", count)
		}
	})

	t.Run("FindAll returns a slice (possibly empty)", func(t *testing.T) {
		results := FindAll[Transform](game.World)
		if results == nil {
			t.Fatal("FindAll returned nil")
		}
	})

	t.Run("FindOne returns a bool indicating if found", func(t *testing.T) {
		result, found := FindOne[Transform](game.World)
		if found {
			// If found, we should have a valid result with an EntityID
			if result.EntityID == 0 {
				t.Fatal("FindOne returned found=true but result.EntityID was zero")
			}
		}
	})
}

// MockSystem implements the System interface for testing
type MockSystem struct {
}

func (m *MockSystem) Init(w *World) error {
	return nil
}

func (m *MockSystem) Update(w *World, delta float64) error {
	return nil
}

func (m *MockSystem) Shutdown(w *World) error {
	return nil
}

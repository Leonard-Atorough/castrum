package castrum

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/internal/camera"
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
		cam, err := game.World.GetComponent[camera.Camera](game.CameraEntityID)
		if err != nil {
			t.Fatalf("Failed to get camera component: %v", err)
		}
		if cam.ScreenSize.X != 320 || cam.ScreenSize.Y != 240 {
			t.Fatalf("Camera.ScreenSize = %v, want {320 240}", cam.ScreenSize)
		}
		if game.Render == nil || game.World == nil || game.Systems == nil {
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
	cam, err := game.World.GetComponent[camera.Camera](game.CameraEntityID)
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

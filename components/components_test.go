package components

import (
	"image/color"
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestTransformComponent(t *testing.T) {
	t.Run("Create New Transform Component with no color", func(t *testing.T) {
		transform := NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, nil)
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
	})

	t.Run("Create New Transform Component with color", func(t *testing.T) {
		transform := NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
		if transform.Color != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
			t.Errorf("Expected color to be red, got %v", transform.Color)
		}
	})

	t.Run("Create New Transform Component with default values", func(t *testing.T) {
		transform := NewTransformWithDefault()
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
		if transform.Color != color.Transparent {
			t.Errorf("Expected color to be transparent, got %v", transform.Color)
		}
	})
}

func TestSpriteComponent(t *testing.T) {
	t.Run("Create New Sprite Component with all fields", func(t *testing.T) {
		sprite := NewSprite("texture.png", PrimitiveKindRectangle, 0, 0, true, nil)
		if sprite.TexturePath != "texture.png" {
			t.Errorf("Expected texture path to be 'texture.png', got %v", sprite.TexturePath)
		}
		if sprite.Primitive != PrimitiveKindRectangle {
			t.Errorf("Expected primitive to be Rectangle, got %v", sprite.Primitive)
		}
		if sprite.Layer != 0 {
			t.Errorf("Expected layer to be 0, got %v", sprite.Layer)
		}
		if sprite.SortOrder != 0 {
			t.Errorf("Expected sort order to be 0, got %v", sprite.SortOrder)
		}
		if !sprite.Visible {
			t.Errorf("Expected visible to be true, got %v", sprite.Visible)
		}
		data := reflect.ValueOf(sprite.Data)
		if !data.IsValid() || data.Kind() != reflect.Struct || data.NumField() != 0 {
			t.Errorf("Expected data to be an empty struct, got %v", sprite.Data)
		}
	})

	t.Run("Create New Sprite with layer greater than 31", func(t *testing.T) {
		sprite := NewSprite("texture.png", PrimitiveKindRectangle, 35, 0, true, nil)
		if sprite.Layer != 31 {
			t.Errorf("Expected layer to be capped at 31, got %v", sprite.Layer)
		}
	})

	t.Run("Create New Sprite with invalid primitive type", func(t *testing.T) {
		sprite := NewSprite("texture.png", 99, 0, 0, true, nil)
		if sprite.Primitive != PrimitiveKindRectangle {
			t.Errorf("Expected primitive to default to Rectangle, got %v", sprite.Primitive)
		}
	})
}

func TestAnimationComponent(t *testing.T) {
	tests := []struct {
		name     string
		clipID   string
		autoplay bool
	}{
		{"With Autoplay", "clip1", true},
		{"Without Autoplay", "clip2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animation := NewAnimation(tt.clipID, tt.autoplay)
			if animation.ClipPath != tt.clipID {
				t.Errorf("Expected clip path to be '%v', got %v", tt.clipID, animation.ClipPath)
			}
			if animation.Playing != tt.autoplay {
				t.Errorf("Expected playing to be %v, got %v", tt.autoplay, animation.Playing)
			}
			if animation.PlaybackSpeed != 1.0 {
				t.Errorf("Expected playback speed to be 1.0, got %v", animation.PlaybackSpeed)
			}
		})
	}
}

type ColliderTestShape struct {
	Width  float64
	Height float64
}

func (ct *ColliderTestShape) BoundingBox() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: 0, Y: 0},
		Max: geom.Vector2{X: ct.Width, Y: ct.Height},
	}
}

func TestColliderComponent(t *testing.T) {
	t.Run("Create New Collider Component with all fields", func(t *testing.T) {
		testShape := &ColliderTestShape{Width: 10, Height: 10}
		collider := NewCollider(testShape, true, true, 0, 1, 3, 6)
		// testing components.Collider fields
		if collider.Shape != testShape {
			t.Errorf("Expected shape to be %v, got %v", testShape, collider.Shape)
		}
		if collider.Active != true {
			t.Errorf("Expected active to be true, got %v", collider.Active)
		}
		if collider.Trigger != true {
			t.Errorf("Expected trigger to be true, got %v", collider.Trigger)
		}
		if collider.Layer != 0 {
			t.Errorf("Expected layer to be 0, got %v", collider.Layer)
		}
		var expectedMask uint32
		expectedMask = (1 << 1) | (1 << 3) | (1 << 6)
		if collider.Mask != expectedMask {
			t.Errorf("Expected mask to be %v, got %v", expectedMask, collider.Mask)
		}
	})

	t.Run("Create New Collider Component with layer field greater than 31", func(t *testing.T) {
		testShape := &ColliderTestShape{Width: 10, Height: 10}
		collider := NewCollider(testShape, true, true, 150, 1)
		if collider.Layer != 31 {
			t.Errorf("Expected layer to be 31, got %v", collider.Layer)
		}
	})

	t.Run("Collider Bounding Box", func(t *testing.T) {
		testShape := &ColliderTestShape{Width: 10, Height: 10}
		collider := NewCollider(testShape, true, true, 0, 1)
		expectedBoundingBox := geom.Rect{
			Min: geom.Vector2{X: 0, Y: 0},
			Max: geom.Vector2{X: 10, Y: 10},
		}
		if collider.BoundingBox() != expectedBoundingBox {
			t.Errorf("Expected bounding box to be %v, got %v", expectedBoundingBox, collider.BoundingBox())
		}
	})

	t.Run("Collider CanCollideWith", func(t *testing.T) {
		shapeA := &ColliderTestShape{Width: 10, Height: 10}
		shapeB := &ColliderTestShape{Width: 5, Height: 5}
		colliderA := NewCollider(shapeA, true, true, 0, 1)
		colliderB := NewCollider(shapeB, true, true, 1, 0)
		if !colliderA.CanCollideWith(&colliderB) {
			t.Errorf("Expected colliderA to be able to collide with colliderB")
		}
		if !colliderB.CanCollideWith(&colliderA) {
			t.Errorf("Expected colliderB to be able to collide with colliderA")
		}
	})
}

func TestTimerComponent(t *testing.T) {
	t.Run("Create Timer Component", func(t *testing.T) {
		timer := NewTimer("test_timer", 12.5, false, false)
		if timer.ID != "test_timer" {
			t.Errorf("Expected timer ID to be 'test_timer', got %v", timer.ID)
		}
		if timer.Duration != 12.5 {
			t.Errorf("Expected timer duration to be 12.5, got %v", timer.Duration)
		}
		if timer.Running != false {
			t.Errorf("Expected timer running to be false, got %v", timer.Running)
		}
		if timer.Once != false {
			t.Errorf("Expected timer once to be false, got %v", timer.Once)
		}
	})

	t.Run("Create Timer Component with non-positive duration", func(t *testing.T) {
		timer := NewTimer("test_timer", 0, false, false)
		if timer.Duration != 1.0 {
			t.Errorf("Expected timer duration to be 1.0, got %v", timer.Duration)
		}
	})

	t.Run("Start, Stop, and Resume Timer", func(t *testing.T) {
		timer := NewTimer("test_timer", 5, false, false)
		timer.Start()
		if !timer.Running || timer.ElapsedTime != 0 {
			t.Errorf("Expected timer to be running with elapsed time 0, got running=%v, elapsedTime=%v", timer.Running, timer.ElapsedTime)
		}
		timer.Stop()
		if timer.Running {
			t.Errorf("Expected timer to be stopped, got running=%v", timer.Running)
		}
		timer.Resume()
		if !timer.Running {
			t.Errorf("Expected timer to be running after resume, got running=%v", timer.Running)
		}
	})
}

// same as above, now for the Camera component
func TestCameraComponent(t *testing.T) {
	t.Run("Create Camera Component", func(t *testing.T) {
		camera := NewCamera(800, 600)
		if camera.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
			t.Errorf("Expected camera screen size to be %v, got %v", geom.Vector2I{X: 800, Y: 600}, camera.ScreenSize)
		}
		if camera.Position != (geom.Vector2{X: 0, Y: 0}) {
			t.Errorf("Expected camera position to be %v, got %v", geom.Vector2{X: 0, Y: 0}, camera.Position)
		}
		if camera.Zoom != 1 {
			t.Errorf("Expected camera zoom to be 1, got %v", camera.Zoom)
		}
		if camera.Bounds != unboundedRect() {
			t.Errorf("Expected camera bounds to be %v, got %v", unboundedRect(), camera.Bounds)
		}
		if camera.Primary != false {
			t.Errorf("Expected camera primary to be false, got %v", camera.Primary)
		}
	})

	t.Run("Create Camera Component with 0 screen size", func(t *testing.T) {
		camera := NewCamera(0, 0)
		if camera.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
			t.Errorf("Expected camera screen size to be %v, got %v", geom.Vector2I{X: 800, Y: 600}, camera.ScreenSize)
		}
	})

	t.Run("Clamp Camera Position", func(t *testing.T) {
		camera := NewCamera(800, 600)
		camera.Bounds = geom.Rect{
			Min: geom.Vector2{X: -800, Y: -600},
			Max: geom.Vector2{X: 800, Y: 600},
		}
		camera.Position = geom.Vector2{X: 800, Y: 600}
		clampedCamera := camera.ClampPosition()
		if clampedCamera.Position.X > camera.Bounds.Max.X || clampedCamera.Position.X < camera.Bounds.Min.X {
			t.Errorf("Expected clamped camera X to be within bounds, got %v", clampedCamera.Position.X)
		}
		if clampedCamera.Position.Y > camera.Bounds.Max.Y || clampedCamera.Position.Y < camera.Bounds.Min.Y {
			t.Errorf("Expected clamped camera Y to be within bounds, got %v", clampedCamera.Position.Y)
		}
		if clampedCamera.Position.X != camera.Bounds.Max.X/2 {
			t.Errorf("Expected clamped camera X to be %v, got %v", camera.Bounds.Max.X/2, clampedCamera.Position.X)
		}
		if clampedCamera.Position.Y != camera.Bounds.Max.Y/2 {
			t.Errorf("Expected clamped camera Y to be %v, got %v", camera.Bounds.Max.Y/2, clampedCamera.Position.Y)
		}
	})

	t.Run("Is World Rect Visible", func(t *testing.T) {
		camera := NewCamera(800, 600)
		camera.Position = geom.Vector2{X: 0, Y: 0}
		camera.Zoom = 1
		worldRect := geom.Rect{
			Min: geom.Vector2{X: -50, Y: -50},
			Max: geom.Vector2{X: 50, Y: 50},
		}
		if !camera.IsWorldRectVisible(worldRect) {
			t.Errorf("Expected world rect to be visible, got not visible")
		}
	})

	t.Run("Aspect Ratio", func(t *testing.T) {
		camera := NewCamera(800, 600)
		expectedAspectRatio := 800.0 / 600.0
		if camera.AspectRatio() != expectedAspectRatio {
			t.Errorf("Expected aspect ratio to be %v, got %v", expectedAspectRatio, camera.AspectRatio())
		}
	})

	t.Run("World To Screen and Screen To World", func(t *testing.T) {
		camera := NewCamera(800, 600)
		camera.Position = geom.Vector2{X: 0, Y: 0}
		camera.Zoom = 1
		worldPos := geom.Vector2{X: 100, Y: 50}
		screenPos := camera.WorldToScreen(worldPos)
		convertedWorldPos := camera.ScreenToWorld(screenPos)
		if convertedWorldPos != worldPos {
			t.Errorf("Expected converted world position to be %v, got %v", worldPos, convertedWorldPos)
		}
	})

	t.Run("SetScreenSize", func(t *testing.T) {
		camera := NewCamera(800, 600)
		camera.SetScreenSize(1024, 768)
		if camera.ScreenSize != (geom.Vector2I{X: 1024, Y: 768}) {
			t.Errorf("Expected camera screen size to be %v, got %v", geom.Vector2I{X: 1024, Y: 768}, camera.ScreenSize)
		}
	})

	// setscreensize called with 0 width and height should not change the screen size
	t.Run("SetScreenSize with 0 width and height", func(t *testing.T) {
		camera := NewCamera(800, 600)
		camera.SetScreenSize(0, 0)
		if camera.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
			t.Errorf("Expected camera screen size to remain %v, got %v", geom.Vector2I{X: 800, Y: 600}, camera.ScreenSize)
		}
	})
}

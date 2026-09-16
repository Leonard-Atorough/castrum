package components

import (
	"image/color"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestTransformComponent(t *testing.T) {
	t.Run("Create New Transform Component with no color", func(t *testing.T) {
		transform := NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 0, Y: 0})
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
		if transform.Origin.X != 0 || transform.Origin.Y != 0 {
			t.Errorf("Expected origin to be (0,0), got (%v,%v)", transform.Origin.X, transform.Origin.Y)
		}
	})

	t.Run("Create New Transform Component with color", func(t *testing.T) {
		transform := NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 0, Y: 0})
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
		if transform.Origin.X != 0 || transform.Origin.Y != 0 {
			t.Errorf("Expected origin to be (0,0), got (%v,%v)", transform.Origin.X, transform.Origin.Y)
		}
	})
}

func TestSpriteComponent(t *testing.T) {
	t.Run("Create New Sprite Component with all fields", func(t *testing.T) {
		sprite, _ := NewSprite("texture.png", "Atlas-1", "sprite-1", PrimitiveKindRectangle, color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
		if sprite.TexturePath != "texture.png" {
			t.Errorf("Expected texture path to be 'texture.png', got %v", sprite.TexturePath)
		}
		if sprite.AtlasID != "Atlas-1" {
			t.Errorf("Expected atlas ID to be 'Atlas-1', got %v", sprite.AtlasID)
		}
		if sprite.RegionName != "sprite-1" {
			t.Errorf("Expected region name to be 'sprite-1', got %v", sprite.RegionName)
		}
		if sprite.Primitive != PrimitiveKindRectangle {
			t.Errorf("Expected primitive to be Rectangle, got %v", sprite.Primitive)
		}
		if sprite.RenderLayer != 0 {
			t.Errorf("Expected layer to be 0, got %v", sprite.RenderLayer)
		}
		if sprite.SortOrder != 0 {
			t.Errorf("Expected sort order to be 0, got %v", sprite.SortOrder)
		}
		if !sprite.Visible {
			t.Errorf("Expected visible to be true, got %v", sprite.Visible)
		}
		if sprite.FlipH {
			t.Errorf("Expected flipH to be false, got %v", sprite.FlipH)
		}
		if sprite.FlipV {
			t.Errorf("Expected flipV to be false, got %v", sprite.FlipV)
		}
		if sprite.Opacity != 1 {
			t.Errorf("Expected opacity to be 1, got %v", sprite.Opacity)
		}
	})

	t.Run("Create New Sprite with layer greater than 31", func(t *testing.T) {
		sprite, _ := NewSprite("texture.png", "", "", PrimitiveKindRectangle, color.White, geom.Vector2{X: 32, Y: 32}, 35, 0, true, false, false, 1, geom.Polygon{})
		if sprite.RenderLayer != 31 {
			t.Errorf("Expected layer to be capped at 31, got %v", sprite.RenderLayer)
		}
	})

	t.Run("Create New Sprite with invalid primitive type", func(t *testing.T) {
		sprite, _ := NewSprite("texture.png", "", "", 99, color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
		if sprite.Primitive != PrimitiveKindRectangle {
			t.Errorf("Expected primitive to default to Rectangle, got %v", sprite.Primitive)
		}
	})

	t.Run("Create New Sprite with atlasID but no regionName", func(t *testing.T) {
		_, err := NewSprite("texture.png", "Atlas-1", "", PrimitiveKindRectangle, color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
		if err == nil {
			t.Errorf("Expected error when atlasID is specified without regionName")
		}
	})

	t.Run("Create New Sprite with regionName but no atlasID", func(t *testing.T) {
		_, err := NewSprite("texture.png", "", "sprite-1", PrimitiveKindRectangle, color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
		if err == nil {
			t.Errorf("Expected error when regionName is specified without atlasID")
		}
	})

	t.Run("Create New Sprite with opacity clamped to 0-1", func(t *testing.T) {
		sprite, _ := NewSprite("texture.png", "", "", PrimitiveKindRectangle, color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, -0.5, geom.Polygon{})
		if sprite.Opacity != 0 {
			t.Errorf("Expected opacity clamped to 0, got %v", sprite.Opacity)
		}
		sprite, _ = NewSprite("texture.png", "", "", PrimitiveKindRectangle, color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1.5, geom.Polygon{})
		if sprite.Opacity != 1 {
			t.Errorf("Expected opacity clamped to 1, got %v", sprite.Opacity)
		}
	})
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
		testShape := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
		collider, _ := NewCollider(testShape, true, true, geom.Vector2{X: 0, Y: 0}, 0, 1, 3, 6)
		if collider.Shape != testShape {
			t.Errorf("Expected shape to be %v, got %v", testShape, collider.Shape)
		}
		if !collider.Active {
			t.Errorf("Expected active to be true, got %v", collider.Active)
		}
		if !collider.Trigger {
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
		testShape := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
		collider, _ := NewCollider(testShape, true, true, geom.Vector2{X: 0, Y: 0}, 150, 1)
		if collider.Layer != 31 {
			t.Errorf("Expected layer to be 31, got %v", collider.Layer)
		}
	})

	t.Run("Collider Bounding Box", func(t *testing.T) {
		testShape := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
		collider, _ := NewCollider(testShape, true, true, geom.Vector2{X: 0, Y: 0}, 0, 1)
		expectedBoundingBox := geom.Rect{
			Min: geom.Vector2{X: 0, Y: 0},
			Max: geom.Vector2{X: 10, Y: 10},
		}
		if collider.BoundingBox() != expectedBoundingBox {
			t.Errorf("Expected bounding box to be %v, got %v", expectedBoundingBox, collider.BoundingBox())
		}
	})

	t.Run("Collider Bounding Box with Offset", func(t *testing.T) {
		testShape := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
		collider, _ := NewCollider(testShape, true, true, geom.Vector2{X: 5, Y: -3}, 0, 1)
		expectedBoundingBox := geom.Rect{
			Min: geom.Vector2{X: 5, Y: -3},
			Max: geom.Vector2{X: 15, Y: 7},
		}
		if collider.BoundingBox() != expectedBoundingBox {
			t.Errorf("Expected bounding box to be %v, got %v", expectedBoundingBox, collider.BoundingBox())
		}
	})

	t.Run("Collider CanCollideWith", func(t *testing.T) {
		shapeA := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
		shapeB := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 5, Y: 5}}
		colliderA, _ := NewCollider(shapeA, true, true, geom.Vector2{X: 0, Y: 0}, 0, 1)
		colliderB, _ := NewCollider(shapeB, true, true, geom.Vector2{X: 0, Y: 0}, 1, 0)
		if !colliderA.CanCollideWith(colliderB) {
			t.Errorf("Expected colliderA to be able to collide with colliderB")
		}
		if !colliderB.CanCollideWith(colliderA) {
			t.Errorf("Expected colliderB to be able to collide with colliderA")
		}
	})

	t.Run("Supported collider shapes", func(t *testing.T) {
		circle, _ := NewCollider(geom.Circle{Radius: 1}, true, false, geom.Vector2{X: 0, Y: 0}, 0)
		if !circle.IsSupportedShape() {
			t.Error("Circle should be supported")
		}
		rect, _ := NewCollider(geom.Rect{}, true, false, geom.Vector2{X: 0, Y: 0}, 0)
		if !rect.IsSupportedShape() {
			t.Error("Rect should be supported")
		}
	})

	t.Run("Unsupported shape returns error", func(t *testing.T) {
		_, err := NewCollider(&ColliderTestShape{Width: 10, Height: 10}, true, false, geom.Vector2{X: 0, Y: 0}, 0)
		if err == nil {
			t.Error("Expected error for unsupported shape")
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
		camera := NewCamera(800, 600, false)
		if camera.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
			t.Errorf("Expected camera screen size to be %v, got %v", geom.Vector2I{X: 800, Y: 600}, camera.ScreenSize)
		}
		if camera.Position != (geom.Vector2{X: 0, Y: 0}) {
			t.Errorf("Expected camera position to be %v, got %v", geom.Vector2{X: 0, Y: 0}, camera.Position)
		}
		if camera.Zoom != 1 {
			t.Errorf("Expected camera zoom to be 1, got %v", camera.Zoom)
		}
		if camera.Bounds != UnboundedRect() {
			t.Errorf("Expected camera bounds to be %v, got %v", UnboundedRect(), camera.Bounds)
		}
		if camera.Primary != false {
			t.Errorf("Expected camera primary to be false, got %v", camera.Primary)
		}
	})

	t.Run("Create Camera Component with 0 screen size", func(t *testing.T) {
		camera := NewCamera(0, 0, false)
		if camera.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
			t.Errorf("Expected camera screen size to be %v, got %v", geom.Vector2I{X: 800, Y: 600}, camera.ScreenSize)
		}
	})

	t.Run("Clamp Camera Position", func(t *testing.T) {
		camera := NewCamera(800, 600, false)
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
		camera := NewCamera(800, 600, false)
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
		camera := NewCamera(800, 600, false)
		expectedAspectRatio := 800.0 / 600.0
		if camera.AspectRatio() != expectedAspectRatio {
			t.Errorf("Expected aspect ratio to be %v, got %v", expectedAspectRatio, camera.AspectRatio())
		}
	})

	t.Run("World To Screen and Screen To World", func(t *testing.T) {
		camera := NewCamera(800, 600, false)
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
		camera := NewCamera(800, 600, false)
		camera.SetScreenSize(1024, 768)
		if camera.ScreenSize != (geom.Vector2I{X: 1024, Y: 768}) {
			t.Errorf("Expected camera screen size to be %v, got %v", geom.Vector2I{X: 1024, Y: 768}, camera.ScreenSize)
		}
	})

	// setscreensize called with 0 width and height should not change the screen size
	t.Run("SetScreenSize with 0 width and height", func(t *testing.T) {
		camera := NewCamera(800, 600, false)
		camera.SetScreenSize(0, 0)
		if camera.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
			t.Errorf("Expected camera screen size to remain %v, got %v", geom.Vector2I{X: 800, Y: 600}, camera.ScreenSize)
		}
	})
}

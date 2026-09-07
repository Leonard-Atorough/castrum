package camera

import (
	"math"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

func TestNewCamera(t *testing.T) {
	c := NewCamera()

	if c.Zoom != 1 {
		t.Fatalf("Zoom = %v, want 1", c.Zoom)
	}
	if !math.IsInf(c.Bounds.Min.X, -1) || !math.IsInf(c.Bounds.Max.X, 1) {
		t.Fatalf("Bounds = %v, want unbounded", c.Bounds)
	}
	if c.Primary {
		t.Fatal("expected new camera to not be Primary by default")
	}
}

func TestCamera_SetScreenSize(t *testing.T) {
	c := NewCamera()
	c.SetScreenSize(800, 600)

	if c.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
		t.Fatalf("ScreenSize = %v, want {800 600}", c.ScreenSize)
	}
}

func TestCamera_WorldToScreenAndBack(t *testing.T) {
	c := NewCamera()
	c.SetScreenSize(800, 600)

	t.Run("world origin maps to screen center when camera is at origin", func(t *testing.T) {
		got := c.WorldToScreen(geom.Vector2{})
		if got != (geom.Vector2{X: 400, Y: 300}) {
			t.Fatalf("WorldToScreen(origin) = %v, want screen center {400 300}", got)
		}
	})

	t.Run("ScreenToWorld is the inverse of WorldToScreen", func(t *testing.T) {
		c.Position = geom.Vector2{X: 120, Y: -45}
		c.Zoom = 2
		world := geom.Vector2{X: 50, Y: -30}

		screen := c.WorldToScreen(world)
		back := c.ScreenToWorld(screen)

		if !vecAlmostEqual(back, world) {
			t.Fatalf("round-trip mismatch: got %v, want %v", back, world)
		}
	})

	t.Run("zoomed out camera (zoom < 1) maps correctly", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Position = geom.Vector2{X: 100, Y: 100}
		c.Zoom = 0.5

		world := geom.Vector2{X: 200, Y: 150}
		screen := c.WorldToScreen(world)
		back := c.ScreenToWorld(screen)

		if !vecAlmostEqual(back, world) {
			t.Fatalf("zoomed-out round-trip failed: got %v, want %v", back, world)
		}
	})

	t.Run("high zoom level converts correctly", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Position = geom.Vector2{X: 0, Y: 0}
		c.Zoom = 10

		world := geom.Vector2{X: 5, Y: -3}
		screen := c.WorldToScreen(world)
		back := c.ScreenToWorld(screen)

		if !vecAlmostEqual(back, world) {
			t.Fatalf("high zoom round-trip failed: got %v, want %v", back, world)
		}
	})
}

func vecAlmostEqual(a, b geom.Vector2) bool {
	const epsilon = 1e-9
	return math.Abs(a.X-b.X) < epsilon && math.Abs(a.Y-b.Y) < epsilon
}

func TestCamera_ViewportBounds(t *testing.T) {
	c := NewCamera()
	c.SetScreenSize(800, 600)
	c.Position = geom.Vector2{X: 100, Y: 50}
	c.Zoom = 2

	got := c.ViewportBounds()
	want := geom.Rect{
		Min: geom.Vector2{X: 100 - 200, Y: 50 - 150},
		Max: geom.Vector2{X: 100 + 200, Y: 50 + 150},
	}
	if !vecAlmostEqual(got.Min, want.Min) || !vecAlmostEqual(got.Max, want.Max) {
		t.Fatalf("ViewportBounds() = %v, want %v", got, want)
	}

	t.Run("viewport with zoom 1", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Position = geom.Vector2{X: 0, Y: 0}
		c.Zoom = 1

		vb := c.ViewportBounds()
		if !vecAlmostEqual(vb.Min, geom.Vector2{X: -400, Y: -300}) {
			t.Fatalf("Min = %v, want {-400 -300}", vb.Min)
		}
		if !vecAlmostEqual(vb.Max, geom.Vector2{X: 400, Y: 300}) {
			t.Fatalf("Max = %v, want {400 300}", vb.Max)
		}
	})
}

func TestCamera_ClampPosition(t *testing.T) {
	t.Run("unbounded camera is never clamped", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Position = geom.Vector2{X: 1e6, Y: -1e6}

		c = c.ClampPosition()

		if c.Position != (geom.Vector2{X: 1e6, Y: -1e6}) {
			t.Fatalf("Position changed under unbounded Bounds: %v", c.Position)
		}
	})

	t.Run("camera is pulled back inside explicit bounds", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Bounds = geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 1000, Y: 1000}}
		c.Position = geom.Vector2{X: -500, Y: 2000}

		c = c.ClampPosition()

		viewport := c.ViewportBounds()
		if viewport.Min.X < c.Bounds.Min.X-epsilonRect || viewport.Max.X > c.Bounds.Max.X+epsilonRect ||
			viewport.Min.Y < c.Bounds.Min.Y-epsilonRect || viewport.Max.Y > c.Bounds.Max.Y+epsilonRect {
			t.Fatalf("viewport %v escapes bounds %v after clamping", viewport, c.Bounds)
		}
	})

	t.Run("clamping preserves other fields", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(100, 100)
		c.Zoom = 2.5
		c.Rotation = 0.5
		c.Primary = true
		c.Bounds = geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 100, Y: 100}}
		c.Position = geom.Vector2{X: -100, Y: -100}

		clamped := c.ClampPosition()

		if clamped.Zoom != 2.5 {
			t.Fatalf("Zoom changed after clamp: %v", clamped.Zoom)
		}
		if clamped.Rotation != 0.5 {
			t.Fatalf("Rotation changed after clamp: %v", clamped.Rotation)
		}
		if !clamped.Primary {
			t.Fatalf("Primary flag changed after clamp")
		}
	})

	t.Run("clamp at exactly the bounds edge", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(200, 200)
		c.Zoom = 1
		c.Bounds = geom.Rect{Min: geom.Vector2{X: -100, Y: -100}, Max: geom.Vector2{X: 100, Y: 100}}
		c.Position = geom.Vector2{X: 0, Y: 0}

		clamped := c.ClampPosition()
		// At center, should be fine
		if clamped.Position != (geom.Vector2{X: 0, Y: 0}) {
			t.Fatalf("Position at center changed: %v", clamped.Position)
		}
	})
}

const epsilonRect = 1e-9

func TestCamera_IsWorldRectVisible(t *testing.T) {
	c := NewCamera()
	c.SetScreenSize(800, 600)

	cases := []struct {
		name string
		rect geom.Rect
		want bool
	}{
		{"overlapping viewport", geom.Rect{Min: geom.Vector2{X: -10, Y: -10}, Max: geom.Vector2{X: 10, Y: 10}}, true},
		{"far outside viewport", geom.Rect{Min: geom.Vector2{X: 10000, Y: 10000}, Max: geom.Vector2{X: 10010, Y: 10010}}, false},
		{"touching edge", geom.Rect{Min: geom.Vector2{X: 399, Y: 299}, Max: geom.Vector2{X: 401, Y: 301}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.IsWorldRectVisible(tc.rect); got != tc.want {
				t.Fatalf("IsWorldRectVisible(%v) = %v, want %v", tc.rect, got, tc.want)
			}
		})
	}
}

func TestCamera_AspectRatio(t *testing.T) {
	c := NewCamera()
	c.SetScreenSize(800, 600)

	if got, want := c.AspectRatio(), 800.0/600.0; got != want {
		t.Fatalf("AspectRatio() = %v, want %v", got, want)
	}

	t.Run("square screen", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(512, 512)
		if got := c.AspectRatio(); got != 1.0 {
			t.Fatalf("Square aspect ratio = %v, want 1.0", got)
		}
	})

	t.Run("wide screen", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(1920, 1080)
		if got, want := c.AspectRatio(), 1920.0/1080.0; got != want {
			t.Fatalf("Wide aspect ratio = %v, want %v", got, want)
		}
	})
}

// System tests

func TestNewSystem(t *testing.T) {
	sys := NewSystem()
	if sys == nil {
		t.Fatal("NewSystem returned nil")
	}
}

func TestSystem_Init(t *testing.T) {
	sys := NewSystem()
	world := core.NewWorld()

	err := sys.Init(world)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

func TestSystem_Shutdown(t *testing.T) {
	sys := NewSystem()
	world := core.NewWorld()

	err := sys.Shutdown(world)
	if err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestSystem_Update_ClampsCamera(t *testing.T) {
	sys := NewSystem()
	world := core.NewWorld()

	// Create a camera with bounds
	cam := NewCamera()
	cam.SetScreenSize(800, 600)
	cam.Bounds = geom.Rect{
		Min: geom.Vector2{X: 0, Y: 0},
		Max: geom.Vector2{X: 1000, Y: 1000},
	}
	// Position it outside bounds
	cam.Position = geom.Vector2{X: -500, Y: 2000}
	cam.Primary = true

	// Create entity with camera
	eid, err := world.CreateWithComponents("camera", cam)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Update the system
	err = sys.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Retrieve camera and check it was clamped
	updated, err := world.GetComponent[Camera](eid.ID)
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}

	viewport := updated.ViewportBounds()
	if viewport.Min.X < updated.Bounds.Min.X-epsilonRect || viewport.Max.X > updated.Bounds.Max.X+epsilonRect ||
		viewport.Min.Y < updated.Bounds.Min.Y-epsilonRect || viewport.Max.Y > updated.Bounds.Max.Y+epsilonRect {
		t.Fatalf("viewport %v escapes bounds %v after system update", viewport, updated.Bounds)
	}
}

func TestSystem_Update_MultipleCamera(t *testing.T) {
	sys := NewSystem()
	world := core.NewWorld()

	// Create first camera with bounds
	cam1 := NewCamera()
	cam1.SetScreenSize(800, 600)
	cam1.Bounds = geom.Rect{
		Min: geom.Vector2{X: 0, Y: 0},
		Max: geom.Vector2{X: 500, Y: 500},
	}
	cam1.Position = geom.Vector2{X: -200, Y: -200}
	cam1.Primary = true

	eid1, err := world.CreateWithComponents("camera1", cam1)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Create second camera with different bounds
	cam2 := NewCamera()
	cam2.SetScreenSize(800, 600)
	cam2.Bounds = geom.Rect{
		Min: geom.Vector2{X: 1000, Y: 1000},
		Max: geom.Vector2{X: 2000, Y: 2000},
	}
	cam2.Position = geom.Vector2{X: 500, Y: 500}
	cam2.Primary = false

	eid2, err := world.CreateWithComponents("camera2", cam2)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Update system - should clamp both cameras
	err = sys.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Check first camera was clamped
	cam1Updated, _ := world.GetComponent[Camera](eid1.ID)
	vp1 := cam1Updated.ViewportBounds()
	if vp1.Min.X < cam1Updated.Bounds.Min.X-epsilonRect {
		t.Fatalf("camera1 viewport min escapes bounds after update")
	}

	// Check second camera was clamped
	cam2Updated, _ := world.GetComponent[Camera](eid2.ID)
	vp2 := cam2Updated.ViewportBounds()
	if vp2.Min.X < cam2Updated.Bounds.Min.X-epsilonRect {
		t.Fatalf("camera2 viewport min escapes bounds after update")
	}
}

func TestSystem_Update_UnboundedCameraNotAffected(t *testing.T) {
	sys := NewSystem()
	world := core.NewWorld()

	// Create unbounded camera (no explicit bounds = infinite)
	cam := NewCamera()
	cam.SetScreenSize(800, 600)
	cam.Position = geom.Vector2{X: 1e6, Y: 1e6}
	cam.Primary = true

	eid, err := world.CreateWithComponents("camera", cam)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	originalPos := cam.Position

	// Update system
	err = sys.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Check position was not modified
	updated, _ := world.GetComponent[Camera](eid.ID)

	if updated.Position != originalPos {
		t.Fatalf("unbounded camera position changed from %v to %v", originalPos, updated.Position)
	}
}

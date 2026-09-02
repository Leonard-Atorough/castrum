package render

import (
	"math"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestNewCamera(t *testing.T) {
	c := NewCamera()

	if c.Zoom != 1 {
		t.Fatalf("Zoom = %v, want 1", c.Zoom)
	}
	if !math.IsInf(c.Bounds.Min.X, -1) || !math.IsInf(c.Bounds.Max.X, 1) {
		t.Fatalf("Bounds = %v, want unbounded", c.Bounds)
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
}

func TestCamera_ClampPosition(t *testing.T) {
	t.Run("unbounded camera is never clamped", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Position = geom.Vector2{X: 1e6, Y: -1e6}

		c.ClampPosition()

		if c.Position != (geom.Vector2{X: 1e6, Y: -1e6}) {
			t.Fatalf("Position changed under unbounded Bounds: %v", c.Position)
		}
	})

	t.Run("camera is pulled back inside explicit bounds", func(t *testing.T) {
		c := NewCamera()
		c.SetScreenSize(800, 600)
		c.Bounds = geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 1000, Y: 1000}}
		c.Position = geom.Vector2{X: -500, Y: 2000}

		c.ClampPosition()

		viewport := c.ViewportBounds()
		if viewport.Min.X < c.Bounds.Min.X-epsilonRect || viewport.Max.Y > c.Bounds.Max.Y+epsilonRect {
			t.Fatalf("viewport %v escapes bounds %v after clamping", viewport, c.Bounds)
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
}

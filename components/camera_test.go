package components

import (
	"math"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestNewCamera(t *testing.T) {
	cam := NewCamera(800, 600, true)
	if cam.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
		t.Errorf("ScreenSize = %v, want (800,600)", cam.ScreenSize)
	}
	if cam.Position != (geom.Vector2{X: 0, Y: 0}) {
		t.Errorf("Position = %v, want (0,0)", cam.Position)
	}
	if cam.Zoom != 1 {
		t.Errorf("Zoom = %v, want 1", cam.Zoom)
	}
	if !cam.Primary {
		t.Errorf("Primary = false, want true")
	}
}

func TestNewCameraDefaultsOnZeroScreenSize(t *testing.T) {
	cam := NewCamera(0, 0, false)
	if cam.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
		t.Errorf("ScreenSize = %v, want (800,600) default", cam.ScreenSize)
	}
}

func TestCameraValidate(t *testing.T) {
	tests := []struct {
		name    string
		cam     Camera
		wantErr bool
	}{
		{"valid", Camera{Zoom: 1, ScreenSize: geom.Vector2I{X: 800, Y: 600}}, false},
		{"zero zoom", Camera{Zoom: 0, ScreenSize: geom.Vector2I{X: 800, Y: 600}}, true},
		{"negative zoom", Camera{Zoom: -1, ScreenSize: geom.Vector2I{X: 800, Y: 600}}, true},
		{"zero screen width", Camera{Zoom: 1, ScreenSize: geom.Vector2I{X: 0, Y: 600}}, true},
		{"zero screen height", Camera{Zoom: 1, ScreenSize: geom.Vector2I{X: 800, Y: 0}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cam.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCameraWorldToScreenScreenToWorldInverse(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.Position = geom.Vector2{X: 100, Y: 50}
	cam.Zoom = 2.0

	worldPos := geom.Vector2{X: 300, Y: 200}
	screenPos := cam.WorldToScreen(worldPos)
	convertedWorldPos := cam.ScreenToWorld(screenPos)

	if math.Abs(convertedWorldPos.X-worldPos.X) > 1e-9 || math.Abs(convertedWorldPos.Y-worldPos.Y) > 1e-9 {
		t.Errorf("round-trip: got %v, want %v", convertedWorldPos, worldPos)
	}
}

func TestCameraViewportBounds(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.Zoom = 2.0

	bounds := cam.ViewportBounds()
	wantHalfW := 800.0 / (2 * 2.0)
	wantHalfH := 600.0 / (2 * 2.0)

	if math.Abs(bounds.Width()-wantHalfW*2) > 1e-9 {
		t.Errorf("viewport width = %v, want %v", bounds.Width(), wantHalfW*2)
	}
	if math.Abs(bounds.Height()-wantHalfH*2) > 1e-9 {
		t.Errorf("viewport height = %v, want %v", bounds.Height(), wantHalfH*2)
	}
}

func TestCameraClampPosition(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.Bounds = geom.Rect{
		Min: geom.Vector2{X: -400, Y: -300},
		Max: geom.Vector2{X: 400, Y: 300},
	}
	cam.Position = geom.Vector2{X: 1000, Y: 1000}

	clamped := cam.ClampPosition()

	viewport := clamped.ViewportBounds()
	if viewport.Min.X < cam.Bounds.Min.X || viewport.Max.X > cam.Bounds.Max.X {
		t.Errorf("viewport X out of bounds: %v within %v", viewport, cam.Bounds)
	}
	if viewport.Min.Y < cam.Bounds.Min.Y || viewport.Max.Y > cam.Bounds.Max.Y {
		t.Errorf("viewport Y out of bounds: %v within %v", viewport, cam.Bounds)
	}
}

func TestCameraClampPositionUnboundedNoOp(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.Position = geom.Vector2{X: 10000, Y: 10000}
	clamped := cam.ClampPosition()
	if clamped.Position != cam.Position {
		t.Errorf("unbounded clamp should not change position: got %v, want %v", clamped.Position, cam.Position)
	}
}

func TestCameraIsWorldRectVisible(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.Zoom = 1

	visible := geom.Rect{Min: geom.Vector2{X: -50, Y: -50}, Max: geom.Vector2{X: 50, Y: 50}}
	if !cam.IsWorldRectVisible(visible) {
		t.Error("rect at origin should be visible")
	}

	offscreen := geom.Rect{Min: geom.Vector2{X: 1000, Y: 1000}, Max: geom.Vector2{X: 1100, Y: 1100}}
	if cam.IsWorldRectVisible(offscreen) {
		t.Error("rect at (1000,1000) should not be visible")
	}
}

func TestCameraAspectRatio(t *testing.T) {
	cam := NewCamera(800, 600, false)
	want := 800.0 / 600.0
	if got := cam.AspectRatio(); got != want {
		t.Errorf("AspectRatio() = %v, want %v", got, want)
	}
}

func TestCameraSetScreenSize(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.SetScreenSize(1024, 768)
	if cam.ScreenSize != (geom.Vector2I{X: 1024, Y: 768}) {
		t.Errorf("ScreenSize = %v, want (1024,768)", cam.ScreenSize)
	}
}

func TestCameraSetScreenSizeRejectsZero(t *testing.T) {
	cam := NewCamera(800, 600, false)
	cam.SetScreenSize(0, 0)
	if cam.ScreenSize != (geom.Vector2I{X: 800, Y: 600}) {
		t.Errorf("ScreenSize = %v, want (800,600) unchanged", cam.ScreenSize)
	}
}

func TestCameraSerializeDeserializeRoundTripUnbounded(t *testing.T) {
	original := NewCamera(800, 600, true)
	original.Position = geom.Vector2{X: 100, Y: 50}
	original.Zoom = 1.5

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Camera{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed.Position != original.Position {
		t.Errorf("Position = %v, want %v", reconstructed.Position, original.Position)
	}
	if reconstructed.Zoom != original.Zoom {
		t.Errorf("Zoom = %v, want %v", reconstructed.Zoom, original.Zoom)
	}
	if reconstructed.ScreenSize != original.ScreenSize {
		t.Errorf("ScreenSize = %v, want %v", reconstructed.ScreenSize, original.ScreenSize)
	}
	if reconstructed.Primary != original.Primary {
		t.Errorf("Primary = %v, want %v", reconstructed.Primary, original.Primary)
	}
	if !isUnbounded(reconstructed.Bounds) {
		t.Error("expected unbounded bounds after round-trip")
	}
}

func TestCameraSerializeDeserializeRoundTripWithBounds(t *testing.T) {
	original := NewCamera(800, 600, false)
	original.Bounds = geom.Rect{
		Min: geom.Vector2{X: -100, Y: -100},
		Max: geom.Vector2{X: 100, Y: 100},
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Camera{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed.Bounds != original.Bounds {
		t.Errorf("Bounds = %v, want %v", reconstructed.Bounds, original.Bounds)
	}
}

func TestCameraSerializeUnboundedBoundsIsNull(t *testing.T) {
	cam := NewCamera(800, 600, false)
	data, err := cam.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	if data["bounds"] != nil {
		t.Errorf("expected bounds = nil for unbounded camera, got %v", data["bounds"])
	}
}

func TestCameraDeserializeMissingBoundsDefaultsToUnbounded(t *testing.T) {
	data := map[string]any{
		"zoom":       float64(1.0),
		"screenSize": map[string]any{"x": float64(800), "y": float64(600)},
		"primary":    true,
	}
	cam, err := Camera{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}
	if !isUnbounded(cam.Bounds) {
		t.Error("expected unbounded bounds when bounds key is absent")
	}
}

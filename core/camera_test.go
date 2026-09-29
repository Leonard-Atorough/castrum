package core

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestCameraValidate(t *testing.T) {
	valid := Camera{Zoom: 2, Primary: true}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Camera rejected: %v", err)
	}
	for name, zoom := range map[string]float64{
		"zero zoom":     0,
		"negative zoom": -1,
	} {
		if err := (Camera{Zoom: zoom}).Validate(); err == nil {
			t.Errorf("Camera with %s should fail validation", name)
		}
	}
}

// A camera is a composed entity: Transform for position, Camera for
// framing. The spawn surface accepts both together.
func TestCameraSpawnsComposedWithTransform(t *testing.T) {
	w := NewWorld()
	entity, err := w.NewEntity(
		Transform{Position: geom.Vector2{X: 100, Y: 50}, Scale: geom.Vector2{X: 1, Y: 1}},
		Camera{Zoom: 1.5, Primary: true},
	)
	if err != nil {
		t.Fatalf("spawn camera entity: %v", err)
	}

	transform, ok := entity.Component[Transform](w)
	if !ok || transform.Position.X != 100 {
		t.Fatalf("camera entity transform = %+v, ok=%v", transform, ok)
	}
	camera, ok := entity.Component[Camera](w)
	if !ok || !camera.Primary || camera.Zoom != 1.5 {
		t.Fatalf("camera entity camera = %+v, ok=%v", camera, ok)
	}
}

func TestWorldToScreen(t *testing.T) {
	for _, tc := range []struct {
		name          string
		world         geom.Vector2
		camera        CameraView
		width, height int
		want          geom.Vector2
	}{
		{
			name:   "camera at origin",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  100, height: 100,
			want: geom.Vector2{X: 50, Y: 50},
		},
		{
			name:   "camera offset",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 10, Y: 20}},
			width:  100, height: 100,
			want: geom.Vector2{X: 40, Y: 30},
		},
		{
			name:   "camera offset with zoom",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: CameraView{Zoom: 2, Position: geom.Vector2{X: 10, Y: 20}},
			width:  100, height: 100,
			want: geom.Vector2{X: 30, Y: 10},
		},
		{
			name:   "world equals camera",
			world:  geom.Vector2{X: 10, Y: 20},
			camera: CameraView{Zoom: 3, Position: geom.Vector2{X: 10, Y: 20}},
			width:  100, height: 100,
			want: geom.Vector2{X: 50, Y: 50},
		},
		{
			name:   "non-square target",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  200, height: 100,
			want: geom.Vector2{X: 100, Y: 50},
		},
		{
			name:   "fractional position rounds to whole pixels",
			world:  geom.Vector2{X: 10.4, Y: 20.4},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  100, height: 100,
			want: geom.Vector2{X: 60, Y: 70},
		},
		{
			name:   "halves round away from zero",
			world:  geom.Vector2{X: 10.5, Y: -0.5},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  100, height: 100,
			want: geom.Vector2{X: 61, Y: 50},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.camera.WorldToScreen(tc.world, tc.width, tc.height)
			if !got.AlmostEqual(tc.want, 1e-9) {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestScreenToWorld(t *testing.T) {
	view := CameraView{Zoom: 2, Position: geom.Vector2{X: 10, Y: 20}}
	got := view.ScreenToWorld(geom.Vector2{X: 30, Y: 10}, 100, 100)
	if !got.AlmostEqual(geom.Vector2{X: 0, Y: 0}, 1e-9) {
		t.Errorf("inverse = %v, want (0, 0)", got)
	}

	fractional := view.ScreenToWorld(geom.Vector2{X: 50.5, Y: 40.25}, 100, 100)
	if !fractional.AlmostEqual(geom.Vector2{X: 10.25, Y: 15.125}, 1e-9) {
		t.Errorf("fractional inverse = %v, want (10.25, 15.125)", fractional)
	}

	for _, screen := range []geom.Vector2{
		{X: 0, Y: 0}, {X: 73, Y: 29}, {X: 100, Y: 100}, {X: 50, Y: 50},
	} {
		world := view.ScreenToWorld(screen, 100, 100)
		if back := view.WorldToScreen(world, 100, 100); back != screen {
			t.Errorf("round trip: screen %v went to %v, want it back", screen, back)
		}
	}
}

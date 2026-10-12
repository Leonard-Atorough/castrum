package render

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestCameraValidate(t *testing.T) {
	for name, cam := range map[string]Camera{
		"zero anchor":         {Primary: true},
		"top-left anchor":     {Anchor: geom.Vector2{X: -0.5, Y: -0.5}, Primary: true},
		"bottom-right anchor": {Anchor: geom.Vector2{X: 0.5, Y: 0.5}, Primary: true},
	} {
		if err := cam.Validate(); err != nil {
			t.Errorf("%s: valid Camera rejected: %v", name, err)
		}
	}
	for name, anchor := range map[string]geom.Vector2{
		"x beyond viewport": {X: 0.75, Y: 0},
		"y beyond viewport": {X: 0, Y: -1},
		"nan anchor":        {X: math.NaN(), Y: 0},
	} {
		if err := (Camera{Anchor: anchor}).Validate(); err == nil {
			t.Errorf("Camera with %s should fail validation", name)
		}
	}
}

// A camera is a composed entity: core.Transform for position and zoom,
// Camera for framing. The spawn surface accepts both together.
func TestCameraSpawnsComposedWithTransform(t *testing.T) {
	w := core.NewWorld()
	entity, err := w.NewEntity(
		core.Transform{Position: geom.Vector2{X: 100, Y: 50}, Scale: geom.Vector2{X: 1.5, Y: 1.5}},
		Camera{Primary: true},
	)
	if err != nil {
		t.Fatalf("spawn camera entity: %v", err)
	}

	transform, ok := entity.Component[core.Transform](w)
	if !ok || transform.Position.X != 100 || transform.Scale.X != 1.5 {
		t.Fatalf("camera entity transform = %+v, ok=%v", transform, ok)
	}
	camera, ok := entity.Component[Camera](w)
	if !ok || !camera.Primary {
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
		{
			name:   "top-left anchor puts the camera position at the corner",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}, Anchor: geom.Vector2{X: -0.5, Y: -0.5}},
			width:  100, height: 100,
			want: geom.Vector2{X: 0, Y: 0},
		},
		{
			name:   "top-left anchor with camera offset",
			world:  geom.Vector2{X: 10, Y: 20},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 10, Y: 20}, Anchor: geom.Vector2{X: -0.5, Y: -0.5}},
			width:  100, height: 100,
			want: geom.Vector2{X: 0, Y: 0},
		},
		{
			name:   "quarter anchor places the position off-center",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}, Anchor: geom.Vector2{X: -0.25, Y: 0.25}},
			width:  100, height: 100,
			want: geom.Vector2{X: 25, Y: 75},
		},
		{
			name:   "rotation moves east above center",
			world:  geom.Vector2{X: 10, Y: 0},
			camera: CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}, Rotation: math.Pi / 2},
			width:  100, height: 100,
			want: geom.Vector2{X: 50, Y: 40},
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

func TestScreenToWorldAnchoredAndRotated(t *testing.T) {
	// Top-left anchor: the anchor point is the corner, so screen (0,0) is
	// exactly the camera position.
	anchored := CameraView{Zoom: 1, Position: geom.Vector2{X: 5, Y: 7}, Anchor: geom.Vector2{X: -0.5, Y: -0.5}}
	if got := anchored.ScreenToWorld(geom.Vector2{X: 0, Y: 0}, 100, 100); !got.AlmostEqual(geom.Vector2{X: 5, Y: 7}, 1e-9) {
		t.Errorf("anchored inverse = %v, want (5, 7)", got)
	}

	// Rotation inverts exactly: screen points map back to the world points
	// their rounded projections came from, within the rounding step.
	rotated := CameraView{Zoom: 2, Position: geom.Vector2{X: 10, Y: 20}, Rotation: 0.7}
	for _, world := range []geom.Vector2{
		{X: 0, Y: 0}, {X: 10, Y: 20}, {X: 33, Y: -17}, {X: -5, Y: 42},
	} {
		screen := rotated.WorldToScreen(world, 100, 100)
		back := rotated.ScreenToWorld(screen, 100, 100)
		if !back.AlmostEqual(world, 0.5/rotated.Zoom+1e-9) {
			t.Errorf("rotated round trip: world %v went to %v", world, back)
		}
	}
}

func TestSpawnCamera(t *testing.T) {
	w := core.NewWorld()
	entity, err := SpawnCamera(w, Camera{Primary: true, Anchor: geom.Vector2{X: -0.5, Y: -0.5}}, geom.Vector2{X: 10, Y: 20}, 2)
	if err != nil {
		t.Fatalf("SpawnCamera: %v", err)
	}
	transform, _ := entity.Component[core.Transform](w)
	if transform.Position != (geom.Vector2{X: 10, Y: 20}) || transform.Scale != (geom.Vector2{X: 2, Y: 2}) {
		t.Errorf("spawned transform = %v, want position (10, 20) and zoom scale 2", transform)
	}
	cam, _ := entity.Component[Camera](w)
	if !cam.Primary || cam.Anchor != (geom.Vector2{X: -0.5, Y: -0.5}) {
		t.Errorf("spawned camera = %+v, want primary with the passed anchor", cam)
	}

	// Zero zoom means 1, so a default spawn renders at unit zoom.
	defaulted, err := SpawnCamera(w, Camera{Primary: true}, geom.Vector2{}, 0)
	if err != nil {
		t.Fatalf("SpawnCamera zero zoom: %v", err)
	}
	transform, _ = defaulted.Component[core.Transform](w)
	if transform.Scale != (geom.Vector2{X: 1, Y: 1}) {
		t.Errorf("zero zoom scale = %v, want 1", transform.Scale)
	}

	for _, tc := range []struct {
		name string
		cam  Camera
		zoom float64
	}{
		{"anchor out of range", Camera{Anchor: geom.Vector2{X: 1}}, 1},
		{"negative zoom", Camera{Primary: true}, -1},
		{"infinite zoom", Camera{Primary: true}, math.Inf(1)},
		{"nan zoom", Camera{Primary: true}, math.NaN()},
	} {
		if _, err := SpawnCamera(w, tc.cam, geom.Vector2{}, tc.zoom); err == nil {
			t.Errorf("SpawnCamera with %s should fail", tc.name)
		}
	}
}

func TestCameraZoom(t *testing.T) {
	// Zero scale components mean 1, so a camera spawned without an explicit
	// scale renders at zoom 1.
	if zoom, err := cameraZoom(geom.Vector2{}, geom.Vector2{}, 0.5); err != nil || zoom != 1 {
		t.Errorf("zero scales: zoom=%v err=%v, want 1 and no error", zoom, err)
	}
	// Interpolation between matching scales is exact at both ends.
	if zoom, err := cameraZoom(geom.Vector2{X: 2, Y: 2}, geom.Vector2{X: 2, Y: 2}, 0.25); err != nil || zoom != 2 {
		t.Errorf("matching scales: zoom=%v err=%v, want 2 and no error", zoom, err)
	}
	// Mid-interpolation lerps toward the current scale.
	if zoom, err := cameraZoom(geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 3, Y: 3}, 0.5); err != nil || zoom != 2 {
		t.Errorf("lerp: zoom=%v err=%v, want 2 and no error", zoom, err)
	}
	for _, tc := range []struct {
		name       string
		prev, curr geom.Vector2
	}{
		{"negative zoom", geom.Vector2{X: -1, Y: -1}, geom.Vector2{X: 1, Y: 1}},
		{"non-uniform zoom", geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 1, Y: 2}},
		{"infinite zoom", geom.Vector2{X: math.Inf(1), Y: math.Inf(1)}, geom.Vector2{X: math.Inf(1), Y: math.Inf(1)}},
		{"nan zoom", geom.Vector2{X: math.NaN(), Y: math.NaN()}, geom.Vector2{X: 1, Y: 1}},
	} {
		if _, err := cameraZoom(tc.prev, tc.curr, 0.5); err == nil {
			t.Errorf("%s should fail validation", tc.name)
		}
	}
}

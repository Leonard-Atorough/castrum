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

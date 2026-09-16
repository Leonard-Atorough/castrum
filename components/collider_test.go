package components

import (
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

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

func TestNewCollider(t *testing.T) {
	shape := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
	collider, err := NewCollider(shape, true, true, geom.Vector2{X: 0, Y: 0}, 0, 1, 3, 6)
	if err != nil {
		t.Fatalf("NewCollider() error = %v", err)
	}
	if collider.Shape != shape {
		t.Errorf("Shape mismatch")
	}
	if !collider.Active || !collider.Trigger {
		t.Errorf("Active=%v Trigger=%v, want both true", collider.Active, collider.Trigger)
	}
	if collider.Layer != 0 {
		t.Errorf("Layer = %d, want 0", collider.Layer)
	}
	wantMask := uint32(1<<1 | 1<<3 | 1<<6)
	if collider.Mask != wantMask {
		t.Errorf("Mask = %v, want %v", collider.Mask, wantMask)
	}
}

func TestNewColliderClampsLayer(t *testing.T) {
	collider, _ := NewCollider(geom.Rect{}, true, false, geom.Vector2{}, 150, 0)
	if collider.Layer != 31 {
		t.Errorf("Layer = %d, want 31 (clamped)", collider.Layer)
	}
}

func TestNewColliderRejectsUnsupportedShape(t *testing.T) {
	_, err := NewCollider(&ColliderTestShape{Width: 10, Height: 10}, true, false, geom.Vector2{}, 0)
	if err == nil {
		t.Error("expected error for unsupported shape")
	}
}

func TestNewColliderRejectsNilShape(t *testing.T) {
	_, err := NewCollider(nil, true, false, geom.Vector2{}, 0)
	if err == nil {
		t.Error("expected error for nil shape")
	}
}

func TestColliderValidate(t *testing.T) {
	tests := []struct {
		name    string
		c       Collider
		wantErr bool
	}{
		{"valid rect", Collider{Shape: geom.Rect{}, Layer: 0}, false},
		{"valid circle", Collider{Shape: geom.Circle{Radius: 1}, Layer: 5}, false},
		{"nil shape", Collider{Shape: nil, Layer: 0}, true},
		{"unsupported shape", Collider{Shape: &ColliderTestShape{}, Layer: 0}, true},
		{"layer > 31", Collider{Shape: geom.Rect{}, Layer: 32}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestColliderBoundingBoxWithOffset(t *testing.T) {
	shape := geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 10, Y: 10}}
	collider, _ := NewCollider(shape, true, true, geom.Vector2{X: 5, Y: -3}, 0)
	want := geom.Rect{Min: geom.Vector2{X: 5, Y: -3}, Max: geom.Vector2{X: 15, Y: 7}}
	if got := collider.BoundingBox(); got != want {
		t.Errorf("BoundingBox() = %v, want %v", got, want)
	}
}

func TestColliderBoundingBoxWithoutOffset(t *testing.T) {
	shape := geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 5, Y: 5}}
	collider, _ := NewCollider(shape, true, false, geom.Vector2{}, 0)
	want := geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 5, Y: 5}}
	if got := collider.BoundingBox(); got != want {
		t.Errorf("BoundingBox() = %v, want %v", got, want)
	}
}

func TestColliderCanCollideWith(t *testing.T) {
	a, _ := NewCollider(geom.Rect{}, true, true, geom.Vector2{}, 0, 1)
	b, _ := NewCollider(geom.Rect{}, true, true, geom.Vector2{}, 1, 0)
	if !a.CanCollideWith(b) {
		t.Error("expected a.CanCollideWith(b) = true")
	}
	if !b.CanCollideWith(a) {
		t.Error("expected b.CanCollideWith(a) = true")
	}
}

func TestColliderCannotCollideWithMismatchedMasks(t *testing.T) {
	a, _ := NewCollider(geom.Rect{}, true, true, geom.Vector2{}, 0)     // mask=0, no collisions
	b, _ := NewCollider(geom.Rect{}, true, true, geom.Vector2{}, 1, 0) // mask=1, collides with layer 0
	if a.CanCollideWith(b) {
		t.Error("expected a.CanCollideWith(b) = false (a has mask 0)")
	}
}

func TestColliderSerializeDeserializeCircleRoundTrip(t *testing.T) {
	original, _ := NewCollider(
		geom.Circle{Center: geom.Vector2{X: 5, Y: 10}, Radius: 3},
		true, false, geom.Vector2{X: 1, Y: 2}, 2, 0,
	)

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Collider{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed.Layer != original.Layer {
		t.Errorf("Layer = %d, want %d", reconstructed.Layer, original.Layer)
	}
	if reconstructed.Mask != original.Mask {
		t.Errorf("Mask = %d, want %d", reconstructed.Mask, original.Mask)
	}
	if reconstructed.Active != original.Active {
		t.Errorf("Active = %v, want %v", reconstructed.Active, original.Active)
	}
	if reconstructed.Trigger != original.Trigger {
		t.Errorf("Trigger = %v, want %v", reconstructed.Trigger, original.Trigger)
	}
	if reconstructed.Offset != original.Offset {
		t.Errorf("Offset = %v, want %v", reconstructed.Offset, original.Offset)
	}
	circle, ok := reconstructed.Shape.(geom.Circle)
	if !ok {
		t.Fatalf("Shape type = %T, want geom.Circle", reconstructed.Shape)
	}
	if circle.Center != (geom.Vector2{X: 5, Y: 10}) {
		t.Errorf("Center = %v, want (5,10)", circle.Center)
	}
	if circle.Radius != 3 {
		t.Errorf("Radius = %v, want 3", circle.Radius)
	}
}

func TestColliderSerializeDeserializeRectRoundTrip(t *testing.T) {
	original, _ := NewCollider(
		geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 5, Y: 5}},
		true, true, geom.Vector2{X: 0, Y: 0}, 0, 1,
	)

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Collider{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	rect, ok := reconstructed.Shape.(geom.Rect)
	if !ok {
		t.Fatalf("Shape type = %T, want geom.Rect", reconstructed.Shape)
	}
	if rect != (geom.Rect{Min: geom.Vector2{X: -5, Y: -5}, Max: geom.Vector2{X: 5, Y: 5}}) {
		t.Errorf("Rect = %v, want (-5,-5)-(5,5)", rect)
	}
}

func TestColliderSerializeRejectsUnsupportedShape(t *testing.T) {
	c := Collider{Shape: &ColliderTestShape{}, Layer: 0}
	_, err := c.Serialize()
	if err == nil {
		t.Error("Serialize() expected error for unsupported shape")
	}
}

func TestColliderDeserializeRejectsInvalidLayer(t *testing.T) {
	data := map[string]any{
		"shape": map[string]any{
			"type": "rect",
			"min":  map[string]any{"x": float64(0), "y": float64(0)},
			"max":  map[string]any{"x": float64(10), "y": float64(10)},
		},
		"layer": float64(32),
	}
	_, err := Collider{}.Deserialize(data)
	if err == nil {
		t.Error("Deserialize() expected error for layer > 31")
	}
}

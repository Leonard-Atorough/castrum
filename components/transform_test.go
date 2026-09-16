package components

import (
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestNewTransform(t *testing.T) {
	tr := NewTransform(
		geom.Vector2{X: 10, Y: 20},
		0.5,
		geom.Vector2{X: 2, Y: 3},
		geom.Vector2{X: -1, Y: 0},
	)
	if tr.Position != (geom.Vector2{X: 10, Y: 20}) {
		t.Errorf("Position = %v, want (10,20)", tr.Position)
	}
	if tr.Rotation != 0.5 {
		t.Errorf("Rotation = %v, want 0.5", tr.Rotation)
	}
	if tr.Scale != (geom.Vector2{X: 2, Y: 3}) {
		t.Errorf("Scale = %v, want (2,3)", tr.Scale)
	}
	if tr.Origin != (geom.Vector2{X: -1, Y: 0}) {
		t.Errorf("Origin = %v, want (-1,0)", tr.Origin)
	}
}

func TestTransformValidate(t *testing.T) {
	tests := []struct {
		name    string
		tr      Transform
		wantErr bool
	}{
		{"valid", Transform{Scale: geom.Vector2{X: 1, Y: 1}}, false},
		{"valid negative scale", Transform{Scale: geom.Vector2{X: -1, Y: -1}}, false},
		{"zero scale X", Transform{Scale: geom.Vector2{X: 0, Y: 1}}, true},
		{"zero scale Y", Transform{Scale: geom.Vector2{X: 1, Y: 0}}, true},
		{"zero scale both", Transform{Scale: geom.Vector2{X: 0, Y: 0}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tr.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransformSerializeDeserializeRoundTrip(t *testing.T) {
	original := Transform{
		Position: geom.Vector2{X: 10, Y: 20},
		Rotation: 0.5,
		Scale:    geom.Vector2{X: 2, Y: 3},
		Origin:   geom.Vector2{X: -1, Y: 0},
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Transform{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed != original {
		t.Errorf("round-trip mismatch:\n  got  = %+v\n  want = %+v", reconstructed, original)
	}
}

func TestTransformDeserializeRejectsZeroScale(t *testing.T) {
	data := map[string]any{
		"scale": map[string]any{"x": float64(0), "y": float64(1)},
	}
	_, err := Transform{}.Deserialize(data)
	if err == nil {
		t.Error("Deserialize() expected error for zero scale, got nil")
	}
}

package core

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestTransformValidate(t *testing.T) {
	valid := Transform{Scale: geom.Vector2{X: 1, Y: 1}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Transform rejected: %v", err)
	}
	for name, scale := range map[string]geom.Vector2{
		"zero x": {X: 0, Y: 1},
		"zero y": {X: 1, Y: 0},
	} {
		if err := (Transform{Scale: scale}).Validate(); err == nil {
			t.Errorf("Transform with %s scale should fail validation", name)
		}
	}
}

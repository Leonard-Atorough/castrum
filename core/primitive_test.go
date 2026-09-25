package core

import (
	"image/color"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestPrimitiveValidate(t *testing.T) {
	valid := Primitive{Layer: 31, Color: color.White, Transparency: 0.5}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Primitive rejected: %v", err)
	}
	outlined := Primitive{Outline: true, StrokeWidth: 2}
	if err := outlined.Validate(); err != nil {
		t.Fatalf("outlined Primitive rejected: %v", err)
	}
	for name, prim := range map[string]Primitive{
		"layer over 31":         {Layer: 32},
		"transparency over one": {Transparency: 1.1},
		"negative transparency": {Transparency: -0.1},
		"negative stroke width": {StrokeWidth: -1},
		"outline without width": {Outline: true},
	} {
		if err := prim.Validate(); err == nil {
			t.Errorf("Primitive with %s should fail validation", name)
		}
	}
}

func TestPrimitiveGeometryVariantsValidate(t *testing.T) {
	if err := (RectPrimitive{Size: geom.Vector2{X: 10, Y: 20}}).Validate(); err != nil {
		t.Fatalf("valid RectPrimitive rejected: %v", err)
	}
	if err := (CirclePrimitive{Radius: 5}).Validate(); err != nil {
		t.Fatalf("valid CirclePrimitive rejected: %v", err)
	}
	if err := (LinePrimitive{To: geom.Vector2{X: 3, Y: 0}}).Validate(); err != nil {
		t.Fatalf("valid LinePrimitive rejected: %v", err)
	}

	for name, bad := range map[string]Validatable{
		"rect zero size":         RectPrimitive{},
		"rect zero width":        RectPrimitive{Size: geom.Vector2{X: 10}},
		"rect negative size":     RectPrimitive{Size: geom.Vector2{X: -1, Y: 10}},
		"circle zero radius":     CirclePrimitive{},
		"circle negative radius": CirclePrimitive{Radius: -5},
		"line zero endpoint":     LinePrimitive{},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("%s should fail validation", name)
		}
	}
}

// Invalid primitives are rejected at spawn through the Validatable
// hook, like every other component, and the error names the type.
func TestPrimitiveValidationAtSpawn(t *testing.T) {
	w := NewWorld()
	if _, err := w.NewEntity(
		RectPrimitive{Size: geom.Vector2{X: 10, Y: 10}},
		Primitive{},
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{},
	); err != nil {
		t.Fatalf("spawn with valid rect primitive: %v", err)
	}

	if _, err := w.NewEntity(
		RectPrimitive{}, // zero size: degenerate
		Primitive{},
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{},
	); err == nil || !strings.Contains(err.Error(), "RectPrimitive") {
		t.Errorf("degenerate rect at spawn = %v, want an error naming RectPrimitive", err)
	}
	if _, err := w.NewEntity(
		RectPrimitive{Size: geom.Vector2{X: 10, Y: 10}},
		Primitive{Outline: true}, // outline without width
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{},
	); err == nil || !strings.Contains(err.Error(), "Primitive") {
		t.Errorf("outline without width at spawn = %v, want an error naming Primitive", err)
	}
}

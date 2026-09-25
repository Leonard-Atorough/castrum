package core

import (
	"image/color"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestPrimitiveValidate(t *testing.T) {
	valid := Primitive{
		Shape:        RectShape{Size: geom.Vector2{X: 10, Y: 20}},
		Layer:        31,
		Color:        color.White,
		Transparency: 0.5,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Primitive rejected: %v", err)
	}
	outlined := Primitive{
		Shape:       CircleShape{Radius: 5},
		Outline:     true,
		StrokeWidth: 2,
	}
	if err := outlined.Validate(); err != nil {
		t.Fatalf("outlined Primitive rejected: %v", err)
	}

	for name, prim := range map[string]Primitive{
		"layer over 31":          {Shape: RectShape{Size: geom.Vector2{X: 1, Y: 1}}, Layer: 32},
		"transparency over one":  {Shape: RectShape{Size: geom.Vector2{X: 1, Y: 1}}, Transparency: 1.1},
		"negative transparency":  {Shape: RectShape{Size: geom.Vector2{X: 1, Y: 1}}, Transparency: -0.1},
		"negative stroke width":  {Shape: RectShape{Size: geom.Vector2{X: 1, Y: 1}}, StrokeWidth: -1},
		"outline without width":  {Shape: RectShape{Size: geom.Vector2{X: 1, Y: 1}}, Outline: true},
		"nil shape":              {},
		"rect zero size":         {Shape: RectShape{}},
		"rect zero width":        {Shape: RectShape{Size: geom.Vector2{X: 10}}},
		"rect negative size":     {Shape: RectShape{Size: geom.Vector2{X: -1, Y: 10}}},
		"circle zero radius":     {Shape: CircleShape{}},
		"circle negative radius": {Shape: CircleShape{Radius: -5}},
		"line zero endpoint":     {Shape: LineShape{}},
	} {
		if err := prim.Validate(); err == nil {
			t.Errorf("Primitive with %s should fail validation", name)
		}
	}
}

// Invalid primitives are rejected at spawn through the Validatable
// hook, like every other component, and the error names the component
// type.
func TestPrimitiveValidationAtSpawn(t *testing.T) {
	w := NewWorld()
	if _, err := w.NewEntity(
		Primitive{Shape: RectShape{Size: geom.Vector2{X: 10, Y: 10}}},
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{},
	); err != nil {
		t.Fatalf("spawn with valid primitive: %v", err)
	}

	if _, err := w.NewEntity(
		Primitive{}, // nil shape: no geometry
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{},
	); err == nil || !strings.Contains(err.Error(), "Primitive") {
		t.Errorf("nil shape at spawn = %v, want an error naming Primitive", err)
	}
	if _, err := w.NewEntity(
		Primitive{Shape: RectShape{}}, // degenerate geometry
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{},
	); err == nil || !strings.Contains(err.Error(), "Primitive") {
		t.Errorf("degenerate rect at spawn = %v, want an error naming Primitive", err)
	}
}

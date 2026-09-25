package core

import (
	"image/color"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestSpriteValidate(t *testing.T) {
	valid := Sprite{
		Drawable:     AtlasSource{Atlas: "sprites", Region: "player"},
		Layer:        31,
		Tint:         color.White,
		Transparency: 0.5,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Sprite rejected: %v", err)
	}
	outlined := Sprite{
		Drawable:    CircleShape{Radius: 5},
		Outline:     true,
		StrokeWidth: 2,
	}
	if err := outlined.Validate(); err != nil {
		t.Fatalf("outlined shape Sprite rejected: %v", err)
	}
	// Style without a picture is legal — it just does not draw.
	if err := (Sprite{}).Validate(); err != nil {
		t.Fatalf("nil Drawable should validate, got: %v", err)
	}

	for name, sprite := range map[string]Sprite{
		"layer over 31":          {Layer: 32},
		"transparency over one":  {Transparency: 1.1},
		"negative transparency":  {Transparency: -0.1},
		"negative stroke width":  {StrokeWidth: -1},
		"outline without width":  {Outline: true},
		"atlas without ID":       {Drawable: AtlasSource{Region: "player"}},
		"atlas without region":   {Drawable: AtlasSource{Atlas: "sprites"}},
		"texture without ID":     {Drawable: TextureSource{}},
		"rect zero size":         {Drawable: RectShape{}},
		"rect zero width":        {Drawable: RectShape{Size: geom.Vector2{X: 10}}},
		"rect negative size":     {Drawable: RectShape{Size: geom.Vector2{X: -1, Y: 10}}},
		"circle zero radius":     {Drawable: CircleShape{}},
		"circle negative radius": {Drawable: CircleShape{Radius: -5}},
		"line zero endpoint":     {Drawable: LineShape{}},
	} {
		if err := sprite.Validate(); err == nil {
			t.Errorf("Sprite with %s should fail validation", name)
		}
	}
}

// The Validatable hook fires through the public surface: spawn rejects
// invalid sprites, naming the component type, and a nil-Drawable
// sprite spawns fine — it simply does not draw.
func TestComponentValidationAtSpawn(t *testing.T) {
	w := NewWorld()

	if _, err := w.NewEntity(
		Sprite{Transparency: 2},
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("invalid transparency at spawn = %v, want an error naming Sprite", err)
	}
	if _, err := w.NewEntity(
		Sprite{Drawable: TextureSource{}},
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("empty texture source at spawn = %v, want an error naming Sprite", err)
	}
	if _, err := w.NewEntity(
		Sprite{Drawable: RectShape{}},
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("degenerate rect at spawn = %v, want an error naming Sprite", err)
	}

	entity, err := w.NewEntity(
		Sprite{}, // style only: legal
		Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)
	if err != nil {
		t.Fatalf("nil-Drawable sprite should spawn: %v", err)
	}
	// Attaching the picture later is the supported flow: validate
	// fires again at the add.
	if err := entity.AddComponent(w, Sprite{Drawable: AtlasSource{Atlas: "sprites", Region: "player"}}); err != nil {
		t.Fatalf("attach drawable later: %v", err)
	}
	if err := entity.AddComponent(w, Sprite{Drawable: TextureSource{}}); err == nil {
		t.Error("overwriting with an invalid drawable should error")
	}
}

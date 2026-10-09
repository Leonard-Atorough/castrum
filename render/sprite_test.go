package render

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestSpriteValidate(t *testing.T) {
	valid := Sprite{
		Drawable: AtlasSource{Atlas: "sprites", Region: "player"},
		Layer:    31,
		Color:    color.White,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Sprite rejected: %v", err)
	}
	outlined := Sprite{
		Drawable:    CircleShape{Radii: geom.Vector2{X: 5, Y: 5}},
		Outline:     true,
		StrokeWidth: 2,
	}
	if err := outlined.Validate(); err != nil {
		t.Fatalf("outlined shape Sprite rejected: %v", err)
	}
	// Text sources validate: real strings and empty ones, which draw
	// nothing like a nil drawable does.
	if err := (Sprite{Drawable: TextSource{Font: "ui/go.ttf", Text: "score: 1", Size: 16}}).Validate(); err != nil {
		t.Fatalf("valid text source rejected: %v", err)
	}
	if err := (Sprite{Drawable: TextSource{Font: "ui/go.ttf", Size: 16}}).Validate(); err != nil {
		t.Fatalf("empty text source rejected: %v", err)
	}

	// Style without a picture is legal — it just does not draw.
	if err := (Sprite{}).Validate(); err != nil {
		t.Fatalf("nil Drawable should validate, got: %v", err)
	}

	for name, sprite := range map[string]Sprite{
		"layer over 31":         {Layer: 32},
		"negative stroke width": {StrokeWidth: -1},
		"outline without width": {Outline: true},
		"atlas without ID":      {Drawable: AtlasSource{Region: "player"}},
		"atlas without region":  {Drawable: AtlasSource{Atlas: "sprites"}},
		"texture without ID":    {Drawable: TextureSource{}},
		"rect zero size":        {Drawable: RectShape{}},
		"rect zero width":       {Drawable: RectShape{Size: geom.Vector2{X: 10}}},
		"rect negative size":    {Drawable: RectShape{Size: geom.Vector2{X: -1, Y: 10}}},
		"circle zero radii":     {Drawable: CircleShape{}},
		"circle zero Y radius":  {Drawable: CircleShape{Radii: geom.Vector2{X: 10}}},
		"circle negative radii": {Drawable: CircleShape{Radii: geom.Vector2{X: -5, Y: 5}}},
		"line zero endpoints":   {Drawable: LineShape{}},
		"line same endpoints":   {Drawable: LineShape{From: geom.Vector2{X: 1, Y: 1}, To: geom.Vector2{X: 1, Y: 1}}},
		"text without font":     {Drawable: TextSource{Text: "hi", Size: 16}},
		"text zero size":        {Drawable: TextSource{Font: "f.ttf", Text: "hi"}},
		"text negative size":    {Drawable: TextSource{Font: "f.ttf", Text: "hi", Size: -16}},
		"text NaN size":         {Drawable: TextSource{Font: "f.ttf", Text: "hi", Size: math.NaN()}},
		"text multiline":        {Drawable: TextSource{Font: "f.ttf", Text: "a\nb", Size: 16}},
	} {
		if err := sprite.Validate(); err == nil {
			t.Errorf("Sprite with %s should fail validation", name)
		}
	}
}

// The core.Validatable hook fires through the public surface: spawn rejects
// invalid sprites, naming the component type, and a nil-Drawable
// sprite spawns fine — it simply does not draw.
func TestComponentValidationAtSpawn(t *testing.T) {
	w := core.NewWorld()

	if _, err := w.NewEntity(
		Sprite{Layer: 32},
		core.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("invalid layer at spawn = %v, want an error naming Sprite", err)
	}
	if _, err := w.NewEntity(
		Sprite{Drawable: TextureSource{}},
		core.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("empty texture source at spawn = %v, want an error naming Sprite", err)
	}
	if _, err := w.NewEntity(
		Sprite{Drawable: RectShape{}},
		core.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("degenerate rect at spawn = %v, want an error naming Sprite", err)
	}

	entity, err := w.NewEntity(
		Sprite{}, // style only: legal
		core.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
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

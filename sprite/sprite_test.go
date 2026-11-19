package sprite

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
		Source: Source{Kind: KindSheet, Asset: "sprites.json", Region: "player"},
		Style:  Style{Layer: 31, Color: &color.RGBA{R: 255, G: 255, B: 255, A: 255}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Sprite rejected: %v", err)
	}
	outlined := Sprite{
		Source: Source{Kind: KindCircle, Radii: geom.Vector2{X: 5, Y: 5}},
		Style:  Style{Outline: true, StrokeWidth: 2},
	}
	if err := outlined.Validate(); err != nil {
		t.Fatalf("outlined shape Sprite rejected: %v", err)
	}
	// Text sources validate: real strings and empty ones, which draw
	// nothing like a nil drawable does.
	if err := (Sprite{Source: Source{Kind: KindText, Asset: "ui/go.ttf", Text: "score: 1", TextSize: 16}}).Validate(); err != nil {
		t.Fatalf("valid text source rejected: %v", err)
	}
	if err := (Sprite{Source: Source{Kind: KindText, Asset: "ui/go.ttf", TextSize: 16}}).Validate(); err != nil {
		t.Fatalf("empty text source rejected: %v", err)
	}

	// Style without a picture is legal — it just does not draw.
	if err := (Sprite{}).Validate(); err != nil {
		t.Fatalf("nil Drawable should validate, got: %v", err)
	}

	for name, sprite := range map[string]Sprite{
		"layer over 31":           {Style: Style{Layer: 32}},
		"negative stroke width":   {Style: Style{StrokeWidth: -1}},
		"NaN stroke width":        {Style: Style{StrokeWidth: math.NaN()}},
		"infinite stroke width":   {Style: Style{StrokeWidth: math.Inf(1)}},
		"outline without width":   {Style: Style{Outline: true}},
		"sheet without reference": {Source: Source{Kind: KindSheet, Region: "player"}},
		"sheet without region":    {Source: Source{Kind: KindSheet, Asset: "sprites.json"}},
		"texture without ID":      {Source: Source{Kind: KindTexture}},
		"rect zero size":          {Source: Source{Kind: KindRect}},
		"rect zero width":         {Source: Source{Kind: KindRect, Rect: geom.Vector2{X: 10}}},
		"rect negative size":      {Source: Source{Kind: KindRect, Rect: geom.Vector2{X: -1, Y: 10}}},
		"rect infinite size":      {Source: Source{Kind: KindRect, Rect: geom.Vector2{X: math.Inf(1), Y: 10}}},
		"circle zero radii":       {Source: Source{Kind: KindCircle}},
		"circle zero Y radius":    {Source: Source{Kind: KindCircle, Radii: geom.Vector2{X: 10}}},
		"circle negative radii":   {Source: Source{Kind: KindCircle, Radii: geom.Vector2{X: -5, Y: 5}}},
		"circle NaN radius":       {Source: Source{Kind: KindCircle, Radii: geom.Vector2{X: math.NaN(), Y: 5}}},
		"line zero endpoints":     {Source: Source{Kind: KindLine}},
		"line same endpoints":     {Source: Source{Kind: KindLine, From: geom.Vector2{X: 1, Y: 1}, To: geom.Vector2{X: 1, Y: 1}}},
		"line infinite endpoint":  {Source: Source{Kind: KindLine, From: geom.Vector2{X: math.Inf(1)}, To: geom.Vector2{X: 1}}},
		"text without font":       {Source: Source{Kind: KindText, Text: "hi", TextSize: 16}},
		"text zero size":          {Source: Source{Kind: KindText, Asset: "f.ttf", Text: "hi"}},
		"text negative size":      {Source: Source{Kind: KindText, Asset: "f.ttf", Text: "hi", TextSize: -16}},
		"text NaN size":           {Source: Source{Kind: KindText, Asset: "f.ttf", Text: "hi", TextSize: math.NaN()}},
		"text multiline":          {Source: Source{Kind: KindText, Asset: "f.ttf", Text: "a\nb", TextSize: 16}},
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
		Sprite{Style: Style{Layer: 32}},
		core.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("invalid layer at spawn = %v, want an error naming Sprite", err)
	}
	if _, err := w.NewEntity(
		Sprite{Source: Source{Kind: KindTexture}},
		core.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	); err == nil || !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("empty texture source at spawn = %v, want an error naming Sprite", err)
	}
	if _, err := w.NewEntity(
		Sprite{Source: Source{Kind: KindRect}},
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
	if err := entity.AddComponent(w, Sprite{Source: Source{Kind: KindSheet, Asset: "sprites.json", Region: "player"}}); err != nil {
		t.Fatalf("attach drawable later: %v", err)
	}
	if err := entity.AddComponent(w, Sprite{Source: Source{Kind: KindTexture}}); err == nil {
		t.Error("overwriting with an invalid drawable should error")
	}
}

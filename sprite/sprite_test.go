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
		Asset:  "sprites.json",
		Region: "player",
		Style:  Style{Layer: 31, Color: &color.RGBA{R: 255, G: 255, B: 255, A: 255}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid sheet sprite rejected: %v", err)
	}
	if err := (Sprite{Asset: "hull.png"}).Validate(); err != nil {
		t.Fatalf("whole-texture sprite rejected: %v", err)
	}
	// Style without a picture is legal - it just does not draw.
	if err := (Sprite{}).Validate(); err != nil {
		t.Fatalf("zero-value sprite rejected: %v", err)
	}
	for name, sprite := range map[string]Sprite{
		"layer over 31":         {Asset: "sprites.json", Style: Style{Layer: 32}},
		"negative stroke width": {Asset: "sprites.json", Style: Style{StrokeWidth: -1}},
		"NaN stroke width":      {Asset: "sprites.json", Style: Style{StrokeWidth: math.NaN()}},
		"infinite stroke width": {Asset: "sprites.json", Style: Style{StrokeWidth: math.Inf(1)}},
		"outline without width": {Asset: "sprites.json", Style: Style{Outline: true}},
		"invalid sheet path":    {Asset: "../escape/../sprites.json", Region: "player"},
		"region with no asset":  {Region: "player"},
	} {
		if err := sprite.Validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestTextValidate(t *testing.T) {
	if err := (Text{Asset: "ui/go.ttf", Text: "score: 1", TextSize: 16}).Validate(); err != nil {
		t.Fatalf("valid text rejected: %v", err)
	}
	// No font, or nothing to say: legal, draws nothing.
	if err := (Text{}).Validate(); err != nil {
		t.Fatalf("zero-value text rejected: %v", err)
	}
	if err := (Text{Text: "hi", TextSize: 16}).Validate(); err != nil {
		t.Fatalf("fontless text rejected: %v", err)
	}
	if err := (Text{Asset: "f.ttf", TextSize: 16}).Validate(); err != nil {
		t.Fatalf("empty text rejected: %v", err)
	}
	for name, text := range map[string]Text{
		"zero size":     {Asset: "f.ttf", Text: "hi"},
		"negative size": {Asset: "f.ttf", Text: "hi", TextSize: -16},
		"NaN size":      {Asset: "f.ttf", Text: "hi", TextSize: math.NaN()},
		"multiline":     {Asset: "f.ttf", Text: "a\nb", TextSize: 16},
		"layer over 31": {Asset: "f.ttf", Text: "hi", TextSize: 16, Style: Style{Layer: 32}},
	} {
		if err := text.Validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestPrimitiveValidate(t *testing.T) {
	outlined := Primitive{
		Shape: ShapeCircle,
		Radii: geom.Vector2{X: 5, Y: 5},
		Style: Style{Outline: true, StrokeWidth: 2},
	}
	if err := outlined.Validate(); err != nil {
		t.Fatalf("outlined circle rejected: %v", err)
	}
	if err := (Primitive{}).Validate(); err != nil {
		t.Fatalf("zero-value primitive rejected: %v", err)
	}
	for name, primitive := range map[string]Primitive{
		"rect zero size":          {Shape: ShapeRect},
		"rect negative size":      {Shape: ShapeRect, Rect: geom.Vector2{X: -1, Y: 5}},
		"rect NaN size":           {Shape: ShapeRect, Rect: geom.Vector2{X: math.NaN(), Y: 5}},
		"circle zero radii":       {Shape: ShapeCircle},
		"circle negative radii":   {Shape: ShapeCircle, Radii: geom.Vector2{X: -5, Y: 5}},
		"circle NaN radii":        {Shape: ShapeCircle, Radii: geom.Vector2{X: 5, Y: math.NaN()}},
		"line same endpoints":     {Shape: ShapeLine, From: geom.Vector2{X: 1, Y: 1}, To: geom.Vector2{X: 1, Y: 1}},
		"line NaN endpoints":      {Shape: ShapeLine, From: geom.Vector2{X: math.NaN(), Y: 1}, To: geom.Vector2{X: 1, Y: 1}},
		"unknown shape":           {Shape: Shape(200)},
		"primitive layer over 31": {Shape: ShapeRect, Rect: geom.Vector2{X: 5, Y: 5}, Style: Style{Layer: 32}},
	} {
		if err := primitive.Validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestSpriteValidatedAtWorldBoundary(t *testing.T) {
	w := core.NewWorld()
	// A region without an asset is configuration nonsense: rejected at
	// spawn, named by the rule.
	if _, err := w.NewEntity(Sprite{Region: "player"}); err == nil {
		t.Fatal("region without asset accepted")
	}
	if _, err := w.NewEntity(Sprite{Asset: "../sheets/sprites.json", Region: "player"}); err == nil {
		t.Fatal("invalid sheet reference accepted")
	}
}

func TestStyleValidatedByEveryComponent(t *testing.T) {
	// Style is shared by all three components; each validates it.
	if err := (Sprite{Asset: "a.png", Style: Style{Layer: 99}}).Validate(); err == nil ||
		!strings.Contains(err.Error(), "layer") {
		t.Errorf("sprite layer bounds: %v", err)
	}
	if err := (Text{Asset: "f.ttf", TextSize: 2, Style: Style{Layer: 99}}).Validate(); err == nil ||
		!strings.Contains(err.Error(), "layer") {
		t.Errorf("text layer bounds: %v", err)
	}
	if err := (Primitive{Shape: ShapeRect, Rect: geom.Vector2{X: 1, Y: 1}, Style: Style{Layer: 99}}).Validate(); err == nil ||
		!strings.Contains(err.Error(), "layer") {
		t.Errorf("primitive layer bounds: %v", err)
	}
}

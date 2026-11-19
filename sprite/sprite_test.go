package sprite

import (
	"encoding/json"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func newSpriteWorld(t *testing.T) *core.World {
	t.Helper()
	return core.NewWorld()
}

func TestValidate(t *testing.T) {
	t.Run("sprite", func(t *testing.T) {
		valid := map[string]Sprite{
			"zero value":    {},
			"whole texture": {Asset: "hull.png"},
			"sheet region":  {Asset: "sprites.json", Region: "player", Style: Style{Layer: 31, Color: &color.RGBA{R: 255, G: 255, B: 255, A: 255}}},
		}
		for name, s := range valid {
			if err := s.Validate(); err != nil {
				t.Errorf("%s rejected: %v", name, err)
			}
		}
		invalid := map[string]Sprite{
			"layer over 31":         {Asset: "sprites.json", Style: Style{Layer: 32}},
			"outline without width": {Asset: "sprites.json", Style: Style{Outline: true}},
			"negative stroke width": {Asset: "sprites.json", Style: Style{StrokeWidth: -1}},
			"NaN stroke width":      {Asset: "sprites.json", Style: Style{StrokeWidth: math.NaN()}},
			"invalid sheet path":    {Asset: "../escape/../sprites.json", Region: "player"},
			"region without asset":  {Region: "player"},
		}
		for name, s := range invalid {
			if err := s.Validate(); err == nil {
				t.Errorf("%s accepted", name)
			}
		}
	})
	t.Run("text", func(t *testing.T) {
		valid := map[string]Text{
			"zero value":   {},
			"no font yet":  {Text: "hi", TextSize: 16},
			"empty text":   {Asset: "f.ttf", TextSize: 16},
			"with content": {Asset: "f.ttf", Text: "score: 1", TextSize: 16},
		}
		for name, x := range valid {
			if err := x.Validate(); err != nil {
				t.Errorf("%s rejected: %v", name, err)
			}
		}
		invalid := map[string]Text{
			"zero size":     {Asset: "f.ttf", Text: "hi"},
			"negative size": {Asset: "f.ttf", Text: "hi", TextSize: -16},
			"NaN size":      {Asset: "f.ttf", Text: "hi", TextSize: math.NaN()},
			"multiline":     {Asset: "f.ttf", Text: "a\nb", TextSize: 16},
			"layer over 31": {Asset: "f.ttf", Text: "hi", TextSize: 16, Style: Style{Layer: 32}},
		}
		for name, x := range invalid {
			if err := x.Validate(); err == nil {
				t.Errorf("%s accepted", name)
			}
		}
	})
	t.Run("primitive", func(t *testing.T) {
		valid := map[string]Primitive{
			"zero value":      {},
			"outlined circle": {Shape: ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}, Style: Style{Outline: true, StrokeWidth: 2}},
		}
		for name, p := range valid {
			if err := p.Validate(); err != nil {
				t.Errorf("%s rejected: %v", name, err)
			}
		}
		invalid := map[string]Primitive{
			"rect zero size":        {Shape: ShapeRect},
			"rect negative size":    {Shape: ShapeRect, Rect: geom.Vector2{X: -1, Y: 5}},
			"rect NaN size":         {Shape: ShapeRect, Rect: geom.Vector2{X: math.NaN(), Y: 5}},
			"circle zero radii":     {Shape: ShapeCircle},
			"circle negative radii": {Shape: ShapeCircle, Radii: geom.Vector2{X: -5, Y: 5}},
			"circle NaN radii":      {Shape: ShapeCircle, Radii: geom.Vector2{X: 5, Y: math.NaN()}},
			"line same endpoints":   {Shape: ShapeLine, From: geom.Vector2{X: 1, Y: 1}, To: geom.Vector2{X: 1, Y: 1}},
			"line NaN endpoints":    {Shape: ShapeLine, From: geom.Vector2{X: math.NaN(), Y: 1}, To: geom.Vector2{X: 1, Y: 1}},
			"unknown shape":         {Shape: Shape(200)},
			"layer over 31":         {Shape: ShapeRect, Rect: geom.Vector2{X: 5, Y: 5}, Style: Style{Layer: 32}},
		}
		for name, p := range invalid {
			if err := p.Validate(); err == nil {
				t.Errorf("%s accepted", name)
			}
		}
	})
}

func TestSpriteValidatedAtWorldBoundary(t *testing.T) {
	w := newSpriteWorld(t)
	// A region without an asset is configuration nonsense: rejected at
	// spawn, named by the rule.
	if _, err := w.NewEntity(Sprite{Region: "player"}); err == nil {
		t.Fatal("region without asset accepted")
	}
	if _, err := w.NewEntity(Sprite{Asset: "../sheets/sprites.json", Region: "player"}); err == nil {
		t.Fatal("invalid sheet reference accepted")
	}
}

func TestWire(t *testing.T) {
	t.Run("sprite", func(t *testing.T) {
		for name, tc := range map[string]struct {
			value Sprite
			wire  string // empty means pin only the round trip
		}{
			"texture": {Sprite{Asset: "tex.png"}, `{"asset":"tex.png","style":{}}`},
			"sheet":   {Sprite{Asset: "sprites.json", Region: "player"}, `{"asset":"sprites.json","region":"player","style":{}}`},
			"styled":  {Sprite{Asset: "sprites.json", Region: "player", Style: Style{Layer: 3}}, ""},
		} {
			data, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("%s: marshal: %v", name, err)
			}
			if tc.wire != "" && string(data) != tc.wire {
				t.Errorf("%s wire = %s, want %s", name, data, tc.wire)
			}
			var back Sprite
			if err := json.Unmarshal(data, &back); err != nil {
				t.Fatalf("%s: unmarshal %s: %v", name, data, err)
			}
			if back.Asset != tc.value.Asset || back.Region != tc.value.Region || !sameStyle(back.Style, tc.value.Style) {
				t.Errorf("%s round trip = %+v, want %+v", name, back, tc.value)
			}
		}
	})
	t.Run("text", func(t *testing.T) {
		for name, tc := range map[string]struct {
			value Text
			wire  string
		}{
			"surface": {Text{Asset: "fonts/go.ttf", Text: "score: 1", TextSize: 16}, `{"asset":"fonts/go.ttf","text":"score: 1","textSize":16,"style":{}}`},
			"tinted":  {Text{Asset: "fonts/go.ttf", Text: "hi", TextSize: 16, Style: Style{Color: &color.RGBA{R: 120, G: 200, B: 255, A: 255}}}, ""},
		} {
			data, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("%s: marshal: %v", name, err)
			}
			if tc.wire != "" && string(data) != tc.wire {
				t.Errorf("%s wire = %s, want %s", name, data, tc.wire)
			}
			var back Text
			if err := json.Unmarshal(data, &back); err != nil {
				t.Fatalf("%s: unmarshal %s: %v", name, data, err)
			}
			if back.Asset != tc.value.Asset || back.Text != tc.value.Text || back.TextSize != tc.value.TextSize || !sameStyle(back.Style, tc.value.Style) {
				t.Errorf("%s round trip = %+v, want %+v", name, back, tc.value)
			}
		}
	})
	t.Run("primitive", func(t *testing.T) {
		for name, tc := range map[string]struct {
			value Primitive
			wire  string
		}{
			"rect": {Primitive{Shape: ShapeRect, Rect: geom.Vector2{X: 10, Y: 20}}, `{"shape":"rect","rect":{"x":10,"y":20},"radii":{"x":0,"y":0},"from":{"x":0,"y":0},"to":{"x":0,"y":0},"style":{}}`},
			"tint": {Primitive{Shape: ShapeCircle, Radii: geom.Vector2{X: 24, Y: 24}, Style: Style{Color: &color.RGBA{R: 100, G: 220, B: 140, A: 255}}}, ""},
			"line": {Primitive{Shape: ShapeLine, From: geom.Vector2{X: -10, Y: 0}, To: geom.Vector2{X: 30, Y: 0}, Style: Style{Outline: true, StrokeWidth: 2}}, ""},
		} {
			data, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("%s: marshal: %v", name, err)
			}
			if tc.wire != "" && string(data) != tc.wire {
				t.Errorf("%s wire = %s, want %s", name, data, tc.wire)
			}
			var back Primitive
			if err := json.Unmarshal(data, &back); err != nil {
				t.Fatalf("%s: unmarshal %s: %v", name, data, err)
			}
			if back.Shape != tc.value.Shape || back.Rect != tc.value.Rect || back.Radii != tc.value.Radii ||
				back.From != tc.value.From || back.To != tc.value.To || !sameStyle(back.Style, tc.value.Style) {
				t.Errorf("%s round trip = %+v, want %+v", name, back, tc.value)
			}
		}

		// Shape wires as a name; unknown names fail decode naming the tag.
		for _, shape := range []Shape{ShapeNone, ShapeRect, ShapeCircle, ShapeLine} {
			data, err := json.Marshal(shape)
			if err != nil {
				t.Fatal(err)
			}
			var back Shape
			if err := json.Unmarshal(data, &back); err != nil || back != shape {
				t.Errorf("shape %v round trip = %v, %v", shape, back, err)
			}
		}
		var p Primitive
		err := json.Unmarshal([]byte(`{"shape":"clown"}`), &p)
		if err == nil || !strings.Contains(err.Error(), "clown") {
			t.Errorf("unknown shape name = %v, want it to name the tag", err)
		}
	})
	t.Run("zero values", func(t *testing.T) {
		// Empty objects decode to valid zero values that draw nothing:
		// the zero-value contract survives the composition.
		var s Sprite
		var x Text
		var p Primitive
		json.Unmarshal([]byte(`{}`), &s)
		json.Unmarshal([]byte(`{}`), &x)
		json.Unmarshal([]byte(`{}`), &p)
		for name, err := range map[string]error{
			"sprite":    s.Validate(),
			"text":      x.Validate(),
			"primitive": p.Validate(),
		} {
			if err != nil {
				t.Errorf("%s zero value rejected: %v", name, err)
			}
		}
	})
}

func TestAuthored(t *testing.T) {
	// Runtime zero values stay legal; Authored is the file-boundary
	// opinion that authored content must contribute something.
	for name, tc := range map[string]struct {
		value   AuthoredVal
		wantErr bool
	}{
		"empty sprite":        {Sprite{}, true},
		"sprite with asset":   {Sprite{Asset: "tex.png"}, false},
		"empty text":          {Text{}, true},
		"text with font":      {Text{Asset: "f.ttf", TextSize: 4}, false},
		"shapeless primitive": {Primitive{}, true},
		"shaped primitive":    {Primitive{Shape: ShapeRect, Rect: geom.Vector2{X: 1, Y: 1}}, false},
	} {
		if err := tc.value.Authored(); (err != nil) != tc.wantErr {
			t.Errorf("%s: Authored() = %v, want error %t", name, err, tc.wantErr)
		}
	}
}

// AuthoredVal lets TestAuthored table-ride the three components.
type AuthoredVal interface {
	Authored() error
}

// sameStyle compares styles, dereferencing Color and treating nil as
// distinct from a color.
func sameStyle(a, b Style) bool {
	if (a.Color == nil) != (b.Color == nil) {
		return false
	}
	if a.Color != nil && *a.Color != *b.Color {
		return false
	}
	return a.Outline == b.Outline && a.StrokeWidth == b.StrokeWidth &&
		a.Layer == b.Layer && a.SortOrder == b.SortOrder &&
		a.Hidden == b.Hidden && a.FlipH == b.FlipH && a.FlipV == b.FlipV
}

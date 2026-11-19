package sprite

import (
	"encoding/json"
	"image/color"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestSpriteJSONRoundTrip(t *testing.T) {
	sprites := []Sprite{
		{},
		{Asset: "tex.png"},
		{Asset: "sprites.json", Region: "player"},
		{Asset: "sprites.json", Region: "player", Style: Style{Layer: 3}},
	}
	for _, s := range sprites {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshal %+v: %v", s, err)
		}
		var back Sprite
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back.Asset != s.Asset || back.Region != s.Region || !sameStyle(back.Style, s.Style) {
			t.Errorf("round trip = %+v, want %+v", back, s)
		}
	}
}

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

func TestShapeJSONRoundTrip(t *testing.T) {
	for _, shape := range []Shape{ShapeNone, ShapeRect, ShapeCircle, ShapeLine} {
		data, err := json.Marshal(shape)
		if err != nil {
			t.Fatalf("marshal %v: %v", shape, err)
		}
		var back Shape
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back != shape {
			t.Errorf("shape round trip = %v, want %v", back, shape)
		}
	}
}

func TestTextJSONRoundTrip(t *testing.T) {
	texts := []Text{
		{},
		{Asset: "fonts/go.ttf", Text: "score: 1", TextSize: 16},
		{Asset: "fonts/go.ttf", Text: "hi", TextSize: 16, Style: Style{Color: &color.RGBA{R: 120, G: 200, B: 255, A: 255}}},
	}
	for _, x := range texts {
		data, err := json.Marshal(x)
		if err != nil {
			t.Fatalf("marshal %+v: %v", x, err)
		}
		var back Text
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back.Asset != x.Asset || back.Text != x.Text || back.TextSize != x.TextSize || !sameStyle(back.Style, x.Style) {
			t.Errorf("round trip = %+v, want %+v", back, x)
		}
	}
}

func TestPrimitiveJSONRoundTrip(t *testing.T) {
	primitives := []Primitive{
		{},
		{Shape: ShapeRect, Rect: geom.Vector2{X: 240, Y: 120}},
		{Shape: ShapeCircle, Radii: geom.Vector2{X: 24, Y: 24}, Style: Style{Color: &color.RGBA{R: 100, G: 220, B: 140, A: 255}}},
		{Shape: ShapeLine, From: geom.Vector2{X: -10, Y: 0}, To: geom.Vector2{X: 30, Y: 0}, Style: Style{Outline: true, StrokeWidth: 2}},
	}
	for _, p := range primitives {
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal %+v: %v", p, err)
		}
		var back Primitive
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back.Shape != p.Shape || back.Rect != p.Rect || back.Radii != p.Radii || back.From != p.From || back.To != p.To || !sameStyle(back.Style, p.Style) {
			t.Errorf("round trip = %+v, want %+v", back, p)
		}
	}
}

func TestComponentWireShapes(t *testing.T) {
	// The wire shapes are the contract. Pin them so renames surface here.
	// Shape marshals as its numeric value: the composition experiment keeps
	// enums numeric, another wire identity to freeze if the design sticks.
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"texture sprite", Sprite{Asset: "tex.png"}, `{"asset":"tex.png","style":{}}`},
		{"sheet sprite", Sprite{Asset: "sprites.json", Region: "player"}, `{"asset":"sprites.json","region":"player","style":{}}`},
		{"text", Text{Asset: "fonts/go.ttf", Text: "score: 1", TextSize: 16}, `{"asset":"fonts/go.ttf","text":"score: 1","textSize":16,"style":{}}`},
		{"rect", Primitive{Shape: ShapeRect, Rect: geom.Vector2{X: 10, Y: 20}}, `{"shape":"rect","rect":{"x":10,"y":20},"radii":{"x":0,"y":0},"from":{"x":0,"y":0},"to":{"x":0,"y":0},"style":{}}`},
	}
	for _, tc := range cases {
		data, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatalf("%s: marshal: %v", tc.name, err)
		}
		// The composed components add no trailing noise; zero-value struct
		// fields may appear because structs cannot be omitted by the
		// standard codec (only Primitive's multiplexed geometry does).
		if string(data) != tc.want {
			t.Errorf("%s wire = %s, want %s", tc.name, data, tc.want)
		}
	}
}

func TestComponentZeroValuesDecode(t *testing.T) {
	// Empty objects decode to valid zero values that draw nothing: the
	// zero-value contract survives the composition.
	for name, into := range map[string]any{
		"sprite":    new(Sprite),
		"text":      new(Text),
		"primitive": new(Primitive),
	} {
		if err := json.Unmarshal([]byte(`{}`), into); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if err := (Sprite{}).Validate(); err != nil {
		t.Errorf("sprite zero value rejected: %v", err)
	}
	if err := (Text{}).Validate(); err != nil {
		t.Errorf("text zero value rejected: %v", err)
	}
	if err := (Primitive{}).Validate(); err != nil {
		t.Errorf("primitive zero value rejected: %v", err)
	}
}

func TestWireRejectsBadJSON(t *testing.T) {
	var s Sprite
	if err := json.Unmarshal([]byte(`{`), &s); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Errorf("sprite bad JSON = %v, want a syntax error", err)
	}
	var p Primitive
	// Shape wires as a name; unknown names fail decode naming the tag.
	if err := json.Unmarshal([]byte(`{"shape":"clown"}`), &p); err == nil || !strings.Contains(err.Error(), "clown") {
		t.Errorf("unknown shape name = %v, want it to name the tag", err)
	}
}

func TestAuthoredGuardsFileContent(t *testing.T) {
	// Runtime zero values stay legal; Authored is the file-boundary
	// opinion that authored content must contribute something.
	if err := (Sprite{}).Authored(); err == nil {
		t.Error("empty authored sprite accepted")
	}
	if err := (Text{}).Authored(); err == nil {
		t.Error("empty authored text accepted")
	}
	if err := (Primitive{}).Authored(); err == nil {
		t.Error("shapeless authored primitive accepted")
	}
	if err := (Sprite{Asset: "tex.png"}).Authored(); err != nil {
		t.Errorf("authored sprite rejected: %v", err)
	}
	if err := (Text{Asset: "f.ttf", TextSize: 4}).Authored(); err != nil {
		t.Errorf("authored text rejected: %v", err)
	}
	if err := (Primitive{Shape: ShapeRect, Rect: geom.Vector2{X: 1, Y: 1}}).Authored(); err != nil {
		t.Errorf("authored primitive rejected: %v", err)
	}
}

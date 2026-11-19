package sprite

import (
	"encoding/json"
	"image/color"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func TestKindRoundTrip(t *testing.T) {
	for _, kind := range []Kind{KindNone, KindTexture, KindSheet, KindText, KindRect, KindCircle, KindLine} {
		data, err := json.Marshal(kind)
		if err != nil {
			t.Fatalf("marshal %v: %v", kind, err)
		}
		var back Kind
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back != kind {
			t.Errorf("kind round trip = %v, want %v", back, kind)
		}
	}
}

func TestKindUnmarshalUnknownName(t *testing.T) {
	var kind Kind
	err := json.Unmarshal([]byte(`"clown"`), &kind)
	if err == nil || !strings.Contains(err.Error(), "clown") {
		t.Fatalf("unknown kind error = %v, want it to name the tag", err)
	}
}

func TestSpriteJSONRoundTrip(t *testing.T) {
	sprites := []Sprite{
		{},
		{Source: Source{Kind: KindTexture, Asset: "tex.png"}},
		{Source: Source{Kind: KindSheet, Asset: "sprites.json", Region: "player"}},
		{Source: Source{Kind: KindText, Asset: "fonts/go.ttf", Text: "score: 1", TextSize: 16}},
		{Source: Source{Kind: KindRect, Rect: geom.Vector2{X: 240, Y: 120}}, Style: Style{Layer: 3, SortOrder: -1}},
		{Source: Source{Kind: KindCircle, Radii: geom.Vector2{X: 24, Y: 24}}, Style: Style{Color: &color.RGBA{R: 100, G: 220, B: 140, A: 255}}},
		{Source: Source{Kind: KindLine, From: geom.Vector2{X: -10, Y: 0}, To: geom.Vector2{X: 30, Y: 0}}, Style: Style{Outline: true, StrokeWidth: 2, FlipH: true}},
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
		compareSprites(t, s, back)
	}
}

func compareSprites(t *testing.T, want, got Sprite) {
	t.Helper()
	if got.Source.Kind != want.Source.Kind {
		t.Errorf("kind = %v, want %v", got.Source.Kind, want.Source.Kind)
	}
	if got.Source.Asset != want.Source.Asset || got.Source.Region != want.Source.Region ||
		got.Source.Text != want.Source.Text || got.Source.TextSize != want.Source.TextSize {
		t.Errorf("source = %+v, want %+v", got.Source, want.Source)
	}
	if got.Source.Rect != want.Source.Rect || got.Source.Radii != want.Source.Radii ||
		got.Source.From != want.Source.From || got.Source.To != want.Source.To {
		t.Errorf("source geometry = %+v, want %+v", got.Source, want.Source)
	}
	if got.Style.Outline != want.Style.Outline || got.Style.StrokeWidth != want.Style.StrokeWidth ||
		got.Style.Layer != want.Style.Layer || got.Style.SortOrder != want.Style.SortOrder ||
		got.Style.Hidden != want.Style.Hidden || got.Style.FlipH != want.Style.FlipH || got.Style.FlipV != want.Style.FlipV {
		t.Errorf("style = %+v, want %+v", got.Style, want.Style)
	}
	switch {
	case want.Style.Color == nil && got.Style.Color == nil:
	case want.Style.Color == nil || got.Style.Color == nil:
		t.Errorf("color = %v, want %v", got.Style.Color, want.Style.Color)
	case *got.Style.Color != *want.Style.Color:
		t.Errorf("color = %v, want %v", *got.Style.Color, *want.Style.Color)
	}
}

func TestSpriteJSONWireShape(t *testing.T) {
	// The wire shape is the contract: kind tags, asset/region fields,
	// and lowercase vector components. Pin it so renames surface here.
	// Zero-value vectors are always present: struct fields cannot be
	// omitted by the standard codec, and the extra keys are harmless.
	const want = `{"source":{"kind":"sheet","asset":"sprites.json","region":"player","rect":{"x":1,"y":2},"radii":{"x":0,"y":0},"from":{"x":0,"y":0},"to":{"x":0,"y":0}},"style":{}}`
	s := Sprite{Source: Source{
		Kind: KindSheet, Asset: "sprites.json", Region: "player",
		Rect: geom.Vector2{X: 1, Y: 2}, // unused by the kind; still on the wire
	}}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Errorf("wire = %s, want %s", data, want)
	}
}

func TestSpriteJSONZeroValue(t *testing.T) {
	var s Sprite
	if err := s.Validate(); err != nil {
		t.Fatalf("zero value rejected: %v", err)
	}
	var fromData Sprite
	if err := json.Unmarshal([]byte(`{}`), &fromData); err != nil {
		t.Fatalf("empty object rejected: %v", err)
	}
	if fromData.Source.Kind != KindNone {
		t.Errorf("absent kind = %v, want KindNone", fromData.Source.Kind)
	}
	if fromData.Style.Color != nil {
		t.Errorf("absent color = %v, want nil", fromData.Style.Color)
	}
	if err := fromData.Validate(); err != nil {
		t.Errorf("decoded zero value rejected: %v", err)
	}
}

func TestSpriteJSONUnknownKindFailsUnmarshalOnly(t *testing.T) {
	var s Sprite
	err := json.Unmarshal([]byte(`{"source":{"kind":"clown"}}`), &s)
	if err == nil || !strings.Contains(err.Error(), "clown") {
		t.Fatalf("unknown kind error = %v, want it to name the tag", err)
	}
	// An unknown kind is a wire-level failure and must not decode
	// silently; semantic rules stay in Validate.
	s2 := Sprite{Source: Source{Kind: Kind(200)}}
	if err2 := json.Unmarshal([]byte(`{"source":{"kind":"texture"}}`), &s2); err2 != nil {
		t.Fatalf("valid decode failed: %v", err2)
	}
	if s2.Source.Kind != KindTexture {
		t.Errorf("kind = %v, want KindTexture", s2.Source.Kind)
	}
	if err2 := s2.Validate(); err2 == nil {
		t.Error("unknown-kind data decodes fine; Validate is the boundary")
	}
	// But a decode that only lacks an asset stays silent here and
	// fails Validate, keeping the boundary error shape.
	s3 := Sprite{Source: Source{Kind: KindTexture}}
	if err3 := json.Unmarshal([]byte(`{"source":{"kind":"texture"}}`), &s3); err3 != nil {
		t.Fatalf("texture decode failed: %v", err3)
	}
	if err3 := s3.Validate(); err3 == nil {
		t.Error("missing asset must fail Validate")
	}
}

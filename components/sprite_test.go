package components

import (
	"image/color"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestNewSprite(t *testing.T) {
	sprite, err := NewSprite("tex.png", "Atlas-1", "region-1", PrimitiveKindRectangle,
		color.White, geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
	if err != nil {
		t.Fatalf("NewSprite() error = %v", err)
	}
	if sprite.TexturePath != "tex.png" {
		t.Errorf("TexturePath = %q, want %q", sprite.TexturePath, "tex.png")
	}
	if sprite.AtlasID != "Atlas-1" {
		t.Errorf("AtlasID = %q, want %q", sprite.AtlasID, "Atlas-1")
	}
	if sprite.RegionName != "region-1" {
		t.Errorf("RegionName = %q, want %q", sprite.RegionName, "region-1")
	}
	if sprite.Primitive != PrimitiveKindRectangle {
		t.Errorf("Primitive = %v, want Rectangle", sprite.Primitive)
	}
	if sprite.RenderLayer != 0 {
		t.Errorf("RenderLayer = %d, want 0", sprite.RenderLayer)
	}
	if sprite.SortOrder != 0 {
		t.Errorf("SortOrder = %d, want 0", sprite.SortOrder)
	}
	if !sprite.Visible {
		t.Errorf("Visible = false, want true")
	}
	if sprite.Opacity != 1 {
		t.Errorf("Opacity = %v, want 1", sprite.Opacity)
	}
}

func TestNewSpriteClampsRenderLayer(t *testing.T) {
	sprite, _ := NewSprite("tex.png", "", "", PrimitiveKindRectangle, color.White,
		geom.Vector2{X: 32, Y: 32}, 35, 0, true, false, false, 1, geom.Polygon{})
	if sprite.RenderLayer != 31 {
		t.Errorf("RenderLayer = %d, want 31 (clamped)", sprite.RenderLayer)
	}
}

func TestNewSpriteClampsInvalidPrimitive(t *testing.T) {
	sprite, _ := NewSprite("tex.png", "", "", 99, color.White,
		geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
	if sprite.Primitive != PrimitiveKindRectangle {
		t.Errorf("Primitive = %v, want Rectangle (defaulted)", sprite.Primitive)
	}
}

func TestNewSpriteClampsOpacity(t *testing.T) {
	sprite, _ := NewSprite("tex.png", "", "", PrimitiveKindRectangle, color.White,
		geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, -0.5, geom.Polygon{})
	if sprite.Opacity != 0 {
		t.Errorf("Opacity = %v, want 0 (clamped)", sprite.Opacity)
	}
	sprite, _ = NewSprite("tex.png", "", "", PrimitiveKindRectangle, color.White,
		geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1.5, geom.Polygon{})
	if sprite.Opacity != 1 {
		t.Errorf("Opacity = %v, want 1 (clamped)", sprite.Opacity)
	}
}

func TestNewSpriteRejectsAtlasWithoutRegion(t *testing.T) {
	_, err := NewSprite("tex.png", "Atlas-1", "", PrimitiveKindRectangle, color.White,
		geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
	if err == nil {
		t.Error("expected error for atlasID without regionName")
	}
}

func TestNewSpriteRejectsRegionWithoutAtlas(t *testing.T) {
	_, err := NewSprite("tex.png", "", "region-1", PrimitiveKindRectangle, color.White,
		geom.Vector2{X: 32, Y: 32}, 0, 0, true, false, false, 1, geom.Polygon{})
	if err == nil {
		t.Error("expected error for regionName without atlasID")
	}
}

func TestSpriteValidate(t *testing.T) {
	tests := []struct {
		name    string
		sprite  Sprite
		wantErr bool
	}{
		{"valid textured", Sprite{TexturePath: "tex.png", Opacity: 1}, false},
		{"valid atlas", Sprite{AtlasID: "a", RegionName: "r", Opacity: 1}, false},
		{"valid primitive", Sprite{Primitive: PrimitiveKindCircle, Opacity: 1}, false},
		{"atlas without region", Sprite{AtlasID: "a", RegionName: "", Opacity: 1}, true},
		{"region without atlas", Sprite{AtlasID: "", RegionName: "r", Opacity: 1}, true},
		{"renderLayer > 31", Sprite{RenderLayer: 32, Opacity: 1}, true},
		{"invalid primitive", Sprite{Primitive: 99, Opacity: 1}, true},
		{"negative opacity", Sprite{Opacity: -0.1}, true},
		{"opacity > 1", Sprite{Opacity: 1.1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sprite.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSpriteSerializeDeserializeRoundTrip(t *testing.T) {
	original, _ := NewSprite("tex.png", "", "", PrimitiveKindRectangle,
		color.RGBA{R: 100, G: 150, B: 200, A: 255}, geom.Vector2{X: 32, Y: 32},
		5, 10, true, true, false, 0.75, geom.Polygon{})

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Sprite{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed.TexturePath != original.TexturePath ||
		reconstructed.AtlasID != original.AtlasID ||
		reconstructed.RegionName != original.RegionName ||
		reconstructed.Primitive != original.Primitive ||
		reconstructed.Size != original.Size ||
		reconstructed.RenderLayer != original.RenderLayer ||
		reconstructed.SortOrder != original.SortOrder ||
		reconstructed.Visible != original.Visible ||
		reconstructed.FlipH != original.FlipH ||
		reconstructed.FlipV != original.FlipV ||
		reconstructed.Opacity != original.Opacity {
		t.Errorf("round-trip mismatch:\n  got  = %+v\n  want = %+v", reconstructed, original)
	}

	// Color is an interface; compare via RGBA(). The deserialized color
	// may be nil if it wasn't set.
	if original.Color != nil && reconstructed.Color != nil {
		r1, g1, b1, a1 := reconstructed.Color.RGBA()
		r2, g2, b2, a2 := original.Color.RGBA()
		if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
			t.Errorf("Color RGBA mismatch: got (%d,%d,%d,%d), want (%d,%d,%d,%d)", r1, g1, b1, a1, r2, g2, b2, a2)
		}
	} else if original.Color != nil && reconstructed.Color == nil {
		t.Error("Color was non-nil but deserialized as nil")
	}
}

func TestSpriteSerializeNilColorOmitted(t *testing.T) {
	sprite := Sprite{TexturePath: "tex.png", Opacity: 1}
	data, err := sprite.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	if _, hasColor := data["color"]; hasColor {
		t.Error("expected color key to be absent when Color is nil")
	}
}

func TestSpriteDeserializeFloat64FromJSON(t *testing.T) {
	data := map[string]any{
		"texturePath": "test.png",
		"primitive":   float64(PrimitiveKindCircle),
		"size":        map[string]any{"x": float64(10), "y": float64(20)},
		"renderLayer": float64(5),
		"sortOrder":   float64(-3),
		"visible":     true,
		"flipH":       true,
		"flipV":       false,
		"opacity":     float64(0.5),
	}

	sprite, err := Sprite{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if sprite.Primitive != PrimitiveKindCircle {
		t.Errorf("Primitive = %v, want Circle", sprite.Primitive)
	}
	if sprite.RenderLayer != 5 {
		t.Errorf("RenderLayer = %d, want 5", sprite.RenderLayer)
	}
	if sprite.SortOrder != -3 {
		t.Errorf("SortOrder = %d, want -3", sprite.SortOrder)
	}
	if sprite.Opacity != 0.5 {
		t.Errorf("Opacity = %v, want 0.5", sprite.Opacity)
	}
	if sprite.Size != (geom.Vector2{X: 10, Y: 20}) {
		t.Errorf("Size = %v, want (10,20)", sprite.Size)
	}
	if !sprite.FlipH {
		t.Error("FlipH = false, want true")
	}
	if sprite.FlipV {
		t.Error("FlipV = true, want false")
	}
}

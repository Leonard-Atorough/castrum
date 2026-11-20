package castrum

import (
	"bytes"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/serialization"
	"github.com/Leonard-Atorough/castrum/sprite"
)

func newEngineTestRegistry(t *testing.T) *serialization.Registry {
	t.Helper()
	world := core.NewWorld()
	codecs := asset.NewCodecs()
	if err := world.Provide(func(*core.World) (*asset.Codecs, error) {
		return codecs, nil
	}); err != nil {
		t.Fatalf("provide codecs: %v", err)
	}
	return serialization.NewRegistry(world)
}

func TestRegisterEngineMapsComponentKeys(t *testing.T) {
	reg := newEngineTestRegistry(t)
	if err := registerEngine(reg); err != nil {
		t.Fatalf("registerEngine: %v", err)
	}

	for key, want := range map[string]reflect.Type{
		TransformKey: reflect.TypeFor[core.Transform](),
		ColliderKey:  reflect.TypeFor[collision.Collider](),
		SpriteKey:    reflect.TypeFor[sprite.Sprite](),
		TextKey:      reflect.TypeFor[sprite.Text](),
		PrimitiveKey: reflect.TypeFor[sprite.Primitive](),
	} {
		typ, ok := reg.ComponentType(key)
		if !ok || typ != want {
			t.Errorf("key %q type = %v, %v; want %v", key, typ, ok, want)
		}
		back, ok := reg.KeyForType(want)
		if !ok || back != key {
			t.Errorf("type %v key = %q, %v; want %q", want, back, ok, key)
		}
	}
}

func TestRegisterEngineComponentRoundTrip(t *testing.T) {
	reg := newEngineTestRegistry(t)
	if err := registerEngine(reg); err != nil {
		t.Fatalf("registerEngine: %v", err)
	}

	for _, tt := range []struct {
		key   string
		value any
	}{
		{TransformKey, core.Transform{Position: geom.Vector2{X: 12, Y: 34}, Rotation: 0.5}},
		{SpriteKey, sprite.Sprite{
			Asset: "sprites.json", Region: "player",
			Style: sprite.Style{Layer: 2, Color: &color.RGBA{R: 100, G: 220, B: 140, A: 255}},
		}},
		{TextKey, sprite.Text{
			Asset: "fonts/go.ttf", Text: "hello, castrum", TextSize: 16,
			Style: sprite.Style{Color: &color.RGBA{R: 25, G: 237, B: 237, A: 255}},
		}},
		{PrimitiveKey, sprite.Primitive{
			Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 32, Y: 16},
			Style: sprite.Style{Outline: true, StrokeWidth: 2},
		}},
	} {
		t.Run(tt.key, func(t *testing.T) {
			var buf bytes.Buffer
			if err := reg.EncodeComponent(tt.key, tt.value, &buf); err != nil {
				t.Fatalf("EncodeComponent: %v", err)
			}
			decoded, err := reg.DecodeComponent(tt.key, buf.Bytes())
			if err != nil {
				t.Fatalf("DecodeComponent: %v", err)
			}
			if !reflect.DeepEqual(decoded, tt.value) {
				t.Fatalf("decoded = %+v, want %+v", decoded, tt.value)
			}
		})
	}
}

func TestRegisterEngineRejectsSecondCall(t *testing.T) {
	reg := newEngineTestRegistry(t)
	if err := registerEngine(reg); err != nil {
		t.Fatalf("registerEngine: %v", err)
	}
	if err := registerEngine(reg); err == nil {
		t.Fatal("second registerEngine on the same registry succeeded")
	}
}

// TestTransformWireShape pins core.Transform's data wire exactly: the
// tags are a contract, and renamed fields break authored files.
func TestTransformWireShape(t *testing.T) {
	reg := newEngineTestRegistry(t)
	if err := registerEngine(reg); err != nil {
		t.Fatalf("registerEngine: %v", err)
	}

	var buf bytes.Buffer
	value := core.Transform{Position: geom.Vector2{X: 12, Y: 34}, Rotation: 0.5}
	if err := reg.EncodeComponent(TransformKey, value, &buf); err != nil {
		t.Fatalf("EncodeComponent: %v", err)
	}
	const want = `{"position":{"x":12,"y":34},"rotation":0.5,"scale":{"x":0,"y":0},"offset":{"x":0,"y":0}}`
	// The codec's encoder writes a trailing newline; the wire value is
	// the JSON object itself.
	if strings.TrimSpace(buf.String()) != want {
		t.Fatalf("wire = %s, want %s", buf.String(), want)
	}
}

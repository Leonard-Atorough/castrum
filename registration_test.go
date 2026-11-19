package castrum

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/serialization"
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

func TestRegisterEngineRegistersEngineComponents(t *testing.T) {
	reg := newEngineTestRegistry(t)
	if err := registerEngine(reg); err != nil {
		t.Fatalf("registerEngine: %v", err)
	}

	if typ, ok := reg.ComponentType(TransformKey); !ok || typ != reflect.TypeFor[core.Transform]() {
		t.Fatalf("TransformKey type = %v, %v", typ, ok)
	}
	if typ, ok := reg.ComponentType(ColliderKey); !ok || typ != reflect.TypeFor[collision.Collider]() {
		t.Fatalf("ColliderKey type = %v, %v", typ, ok)
	}
	if key, ok := reg.KeyForType(reflect.TypeFor[core.Transform]()); !ok || key != TransformKey {
		t.Fatalf("Transform key = %q, %v", key, ok)
	}
	if key, ok := reg.KeyForType(reflect.TypeFor[collision.Collider]()); !ok || key != ColliderKey {
		t.Fatalf("Collider key = %q, %v", key, ok)
	}
}

func TestRegisterEngineTransformRoundTrip(t *testing.T) {
	reg := newEngineTestRegistry(t)
	if err := registerEngine(reg); err != nil {
		t.Fatalf("registerEngine: %v", err)
	}

	var buf bytes.Buffer
	if err := reg.EncodeComponent(TransformKey, core.Transform{}, &buf); err != nil {
		t.Fatalf("EncodeComponent: %v", err)
	}
	got, err := reg.DecodeComponent(TransformKey, buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeComponent: %v", err)
	}
	if _, ok := got.(core.Transform); !ok {
		t.Fatalf("decoded %T, want core.Transform", got)
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

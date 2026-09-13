package assets

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
)

type testAsset struct {
	Value string
}

func TestRegistryRejectsDuplicatesAndSupportsOverrides(t *testing.T) {
	registry := newDecoderRegistry()
	typ := reflect.TypeFor[testAsset]()
	first := func(context.Context, io.Reader) (any, error) { return testAsset{Value: "first"}, nil }
	second := func(context.Context, io.Reader) (any, error) { return testAsset{Value: "second"}, nil }

	if err := registry.register(typ, "test", first, false); err != nil {
		t.Fatalf("initial register failed: %v", err)
	}
	if err := registry.register(typ, "test", second, false); err == nil {
		t.Fatal("duplicate registration succeeded")
	}
	if err := registry.register(typ, "test", second, true); err != nil {
		t.Fatalf("override failed: %v", err)
	}
	decoder, ok := registry.lookup(typ, "test")
	if !ok {
		t.Fatal("overridden decoder was not found")
	}
	value, err := decoder(context.Background(), strings.NewReader(""))
	if err != nil || value.(testAsset).Value != "second" {
		t.Fatalf("lookup returned %#v, %v", value, err)
	}
}

func TestCacheSeparatesLoadKeysAndInvalidatesByID(t *testing.T) {
	c := newCache()
	typ := reflect.TypeFor[testAsset]()
	otherType := reflect.TypeFor[string]()
	c.put(NewLoadKey("asset", typ, "test"), testAsset{Value: "typed"})
	c.put(NewLoadKey("asset", otherType, "test"), "other")

	if value, ok := c.get(NewLoadKey("asset", typ, "test")); !ok || value.(testAsset).Value != "typed" {
		t.Fatalf("typed cache lookup = %#v, %v", value, ok)
	}
	c.invalidate("asset")
	if _, ok := c.get(NewLoadKey("asset", typ, "test")); ok {
		t.Fatal("typed cache entry survived invalidation")
	}
	if _, ok := c.get(NewLoadKey("asset", otherType, "test")); ok {
		t.Fatal("other cache entry survived invalidation")
	}
}

func TestServiceDecodeAndEncodeValidateTypes(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()
	if err := service.RegisterDecoder(typ, "test", func(context.Context, io.Reader) (any, error) {
		return testAsset{Value: "decoded"}, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}
	if err := service.RegisterEncoder(typ, "test", func(_ context.Context, writer io.Writer, value any) error {
		_, err := io.WriteString(writer, value.(testAsset).Value)
		return err
	}, false); err != nil {
		t.Fatalf("RegisterEncoder failed: %v", err)
	}

	decoded, err := service.Decode(context.Background(), typ, "test", strings.NewReader(""))
	if err != nil || decoded.(testAsset).Value != "decoded" {
		t.Fatalf("Decode returned %#v, %v", decoded, err)
	}
	var output bytes.Buffer
	if err := service.Encode(context.Background(), typ, "test", &output, testAsset{Value: "encoded"}); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if output.String() != "encoded" {
		t.Fatalf("encoded output = %q, want %q", output.String(), "encoded")
	}
	if _, err := service.Decode(context.Background(), reflect.TypeFor[string](), "missing", strings.NewReader("")); err == nil {
		t.Fatal("missing decoder did not fail")
	}
}

func TestServiceRejectsInvalidCodecResultsAndInputs(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()
	if err := service.RegisterDecoder(typ, "wrong", func(context.Context, io.Reader) (any, error) {
		return "wrong type", nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}
	if _, err := service.Decode(context.Background(), typ, "wrong", strings.NewReader("")); err == nil {
		t.Fatal("Decode accepted a value of the wrong type")
	}

	if err := service.RegisterDecoder(typ, "nil", func(context.Context, io.Reader) (any, error) {
		return nil, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}
	if _, err := service.Decode(context.Background(), typ, "nil", strings.NewReader("")); err == nil {
		t.Fatal("Decode accepted nil for a non-pointer type")
	}

	if err := service.RegisterEncoder(typ, "test", func(context.Context, io.Writer, any) error { return nil }, false); err != nil {
		t.Fatalf("RegisterEncoder failed: %v", err)
	}
	if err := service.Encode(context.Background(), typ, "missing", io.Discard, testAsset{}); err == nil {
		t.Fatal("Encode succeeded without a registered encoder")
	}
	if err := service.Encode(context.Background(), typ, "test", io.Discard, nil); err == nil {
		t.Fatal("Encode accepted nil")
	}
	if err := service.Encode(context.Background(), typ, "test", io.Discard, "wrong type"); err == nil {
		t.Fatal("Encode accepted a value of the wrong type")
	}
}

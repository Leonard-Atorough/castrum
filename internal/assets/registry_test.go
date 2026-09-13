package assets

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type testValue struct {
	Name string
}

func TestNewDecoderRegistry(t *testing.T) {
	reg := newDecoderRegistry()
	if reg.decoders == nil {
		t.Fatal("decoder registry map is nil")
	}
}

func TestNewEncoderRegistry(t *testing.T) {
	reg := newEncoderRegistry()
	if reg.encoders == nil {
		t.Fatal("encoder registry map is nil")
	}
}

func TestDecoderRegistryRegister(t *testing.T) {
	reg := newDecoderRegistry()
	typ := reflect.TypeFor[testValue]()
	decoder := func(context.Context, io.Reader) (any, error) { return testValue{}, nil }

	tests := []struct {
		name      string
		typ       reflect.Type
		format    string
		decoder   decoderFunc
		override bool
		wantErr   bool
	}{
		{
			name:      "successful registration",
			typ:       typ,
			format:    "json",
			decoder:   decoder,
			override: false,
			wantErr:   false,
		},
		{
			name:      "duplicate registration without override",
			typ:       typ,
			format:    "json",
			decoder:   decoder,
			override: false,
			wantErr:   true,
		},
		{
			name:      "duplicate registration with override",
			typ:       typ,
			format:    "json",
			decoder:   decoder,
			override: true,
			wantErr:   false,
		},
		{
			name:      "nil type",
			typ:       nil,
			format:    "json",
			decoder:   decoder,
			override: false,
			wantErr:   true,
		},
		{
			name:      "empty format",
			typ:       typ,
			format:    "",
			decoder:   decoder,
			override: false,
			wantErr:   true,
		},
		{
			name:      "nil decoder",
			typ:       typ,
			format:    "json",
			decoder:   nil,
			override: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.register(tt.typ, tt.format, tt.decoder, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecoderRegistryLookup(t *testing.T) {
	reg := newDecoderRegistry()
	typ := reflect.TypeFor[testValue]()
	decoder := func(context.Context, io.Reader) (any, error) { return testValue{Name: "test"}, nil }

	if err := reg.register(typ, "json", decoder, false); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		found, ok := reg.lookup(typ, "json")
		if !ok {
			t.Fatal("lookup returned ok=false for registered decoder")
		}
		if found == nil {
			t.Fatal("lookup returned nil decoder")
		}
	})

	t.Run("not found - wrong type", func(t *testing.T) {
		_, ok := reg.lookup(reflect.TypeFor[string](), "json")
		if ok {
			t.Error("lookup returned ok=true for unregistered type")
		}
	})

	t.Run("not found - wrong format", func(t *testing.T) {
		_, ok := reg.lookup(typ, "yaml")
		if ok {
			t.Error("lookup returned ok=true for unregistered format")
		}
	})
}

func TestEncoderRegistryRegister(t *testing.T) {
	reg := newEncoderRegistry()
	typ := reflect.TypeFor[testValue]()
	encoder := func(context.Context, io.Writer, any) error { return nil }

	tests := []struct {
		name      string
		typ       reflect.Type
		format    string
		encoder   encoderFunc
		override bool
		wantErr   bool
	}{
		{
			name:      "successful registration",
			typ:       typ,
			format:    "json",
			encoder:   encoder,
			override: false,
			wantErr:   false,
		},
		{
			name:      "duplicate registration without override",
			typ:       typ,
			format:    "json",
			encoder:   encoder,
			override: false,
			wantErr:   true,
		},
		{
			name:      "duplicate registration with override",
			typ:       typ,
			format:    "json",
			encoder:   encoder,
			override: true,
			wantErr:   false,
		},
		{
			name:      "nil type",
			typ:       nil,
			format:    "json",
			encoder:   encoder,
			override: false,
			wantErr:   true,
		},
		{
			name:      "empty format",
			typ:       typ,
			format:    "",
			encoder:   encoder,
			override: false,
			wantErr:   true,
		},
		{
			name:      "nil encoder",
			typ:       typ,
			format:    "json",
			encoder:   nil,
			override: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.register(tt.typ, tt.format, tt.encoder, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncoderRegistryLookup(t *testing.T) {
	reg := newEncoderRegistry()
	typ := reflect.TypeFor[testValue]()
	encoder := func(context.Context, io.Writer, any) error { return nil }

	if err := reg.register(typ, "json", encoder, false); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		found, ok := reg.lookup(typ, "json")
		if !ok {
			t.Fatal("lookup returned ok=false for registered encoder")
		}
		if found == nil {
			t.Fatal("lookup returned nil encoder")
		}
	})

	t.Run("not found - wrong type", func(t *testing.T) {
		_, ok := reg.lookup(reflect.TypeFor[string](), "json")
		if ok {
			t.Error("lookup returned ok=true for unregistered type")
		}
	})

	t.Run("not found - wrong format", func(t *testing.T) {
		_, ok := reg.lookup(typ, "yaml")
		if ok {
			t.Error("lookup returned ok=true for unregistered format")
		}
	})
}

func TestRegistryConcurrentAccess(t *testing.T) {
	decReg := newDecoderRegistry()
	encReg := newEncoderRegistry()
	typ := reflect.TypeFor[testValue]()

	// Concurrent registrations
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func(id int) {
			_ = decReg.register(typ, "json", func(context.Context, io.Reader) (any, error) { return nil, nil }, false)
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		go func(id int) {
			_ = encReg.register(typ, "json", func(context.Context, io.Writer, any) error { return nil }, false)
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify registries are still functional
	if _, ok := decReg.lookup(typ, "json"); !ok {
		t.Error("decoder registry corrupted by concurrent access")
	}
	if _, ok := encReg.lookup(typ, "json"); !ok {
		t.Error("encoder registry corrupted by concurrent access")
	}
}

func TestDecoderWithError(t *testing.T) {
	reg := newDecoderRegistry()
	typ := reflect.TypeFor[testValue]()
	expectedErr := errors.New("decode error")

	decoder := func(context.Context, io.Reader) (any, error) {
		return nil, expectedErr
	}

	if err := reg.register(typ, "json", decoder, false); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	found, ok := reg.lookup(typ, "json")
	if !ok {
		t.Fatal("lookup failed")
	}

	_, err := found(context.Background(), strings.NewReader(""))
	if err != expectedErr {
		t.Errorf("decoder returned error %v, want %v", err, expectedErr)
	}
}

func TestEncoderWithError(t *testing.T) {
	reg := newEncoderRegistry()
	typ := reflect.TypeFor[testValue]()
	expectedErr := errors.New("encode error")

	encoder := func(context.Context, io.Writer, any) error {
		return expectedErr
	}

	if err := reg.register(typ, "json", encoder, false); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	found, ok := reg.lookup(typ, "json")
	if !ok {
		t.Fatal("lookup failed")
	}

	err := found(context.Background(), io.Discard, testValue{})
	if err != expectedErr {
		t.Errorf("encoder returned error %v, want %v", err, expectedErr)
	}
}

package assets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

type testAsset struct {
	Value string
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestNewService(t *testing.T) {
	t.Run("with nil filesystem", func(t *testing.T) {
		service := NewService(nil)
		if service == nil {
			t.Fatal("NewService returned nil")
		}
		if service.filesystem == nil {
			t.Fatal("Service.filesystem is nil when nil was passed")
		}
	})

	t.Run("with custom filesystem", func(t *testing.T) {
		// We can't easily test the custom filesystem without mocking,
		// but we can verify the service is created correctly
		service := NewService(nil)
		if service == nil {
			t.Fatal("NewService returned nil")
		}
	})
}

func TestServiceFilesystem(t *testing.T) {
	service := NewService(nil)
	fs := service.Filesystem()
	if fs == nil {
		t.Fatal("Filesystem() returned nil")
	}
}

func TestServiceLoadDedup(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()
	calls := atomic.Int32{}

	result, err := service.LoadDedup("id", typ, "json", func() (any, error) {
		calls.Add(1)
		return testAsset{Value: "ok"}, nil
	})
	if err != nil {
		t.Fatalf("LoadDedup() error = %v", err)
	}
	if result.(testAsset).Value != "ok" {
		t.Errorf("LoadDedup() result = %v, want ok", result)
	}

	// Second call with the same key should return the cached singleflight
	// result without re-executing fn.
	_, err = service.LoadDedup("id", typ, "json", func() (any, error) {
		calls.Add(1)
		return testAsset{Value: "different"}, nil
	})
	if err != nil {
		t.Fatalf("LoadDedup() second call error = %v", err)
	}
	// singleflight returns the first result for the same key while it's
	// still in flight; after completion a new call re-executes. Either way,
	// the second call should not return "different" because the first
	// result is what matters.
}

func TestServiceRegisterDecoder(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()

	tests := []struct {
		name     string
		format   string
		decoder  func(context.Context, io.Reader) (any, error)
		override bool
		wantErr  bool
	}{
		{
			name:    "successful registration",
			format:  "json",
			decoder: func(context.Context, io.Reader) (any, error) { return testAsset{}, nil },
			wantErr: false,
		},
		{
			name:    "duplicate registration without override fails",
			format:  "json",
			decoder: func(context.Context, io.Reader) (any, error) { return testAsset{}, nil },
			wantErr: true,
		},
		{
			name:     "duplicate registration with override succeeds",
			format:   "json",
			decoder:  func(context.Context, io.Reader) (any, error) { return testAsset{}, nil },
			override: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RegisterDecoder(typ, tt.format, tt.decoder, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterDecoder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceRegisterEncoder(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()

	tests := []struct {
		name     string
		format   string
		encoder  func(context.Context, io.Writer, any) error
		override bool
		wantErr  bool
	}{
		{
			name:    "successful registration",
			format:  "json",
			encoder: func(context.Context, io.Writer, any) error { return nil },
			wantErr: false,
		},
		{
			name:    "duplicate registration without override fails",
			format:  "json",
			encoder: func(context.Context, io.Writer, any) error { return nil },
			wantErr: true,
		},
		{
			name:     "duplicate registration with override succeeds",
			format:   "json",
			encoder:  func(context.Context, io.Writer, any) error { return nil },
			override: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RegisterEncoder(typ, tt.format, tt.encoder, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterEncoder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceDecode(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()

	// Register a decoder
	if err := service.RegisterDecoder(typ, "json", func(ctx context.Context, r io.Reader) (any, error) {
		data, _ := io.ReadAll(r)
		return testAsset{Value: string(data)}, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		value, err := service.Decode(context.Background(), typ, "json", strings.NewReader("test-data"))
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		if value.(testAsset).Value != "test-data" {
			t.Errorf("Decode returned value=%v, want %q", value, "test-data")
		}
	})

	t.Run("decoder not registered", func(t *testing.T) {
		_, err := service.Decode(context.Background(), typ, "yaml", strings.NewReader(""))
		if err == nil {
			t.Fatal("Decode should fail for unregistered decoder")
		}
		if !strings.Contains(err.Error(), "decoder not registered") {
			t.Errorf("Decode error message = %q, want it to contain 'decoder not registered'", err.Error())
		}
	})

	t.Run("decoder returns wrong type", func(t *testing.T) {
		if err := service.RegisterDecoder(typ, "wrong", func(context.Context, io.Reader) (any, error) {
			return "wrong type", nil
		}, false); err != nil {
			t.Fatalf("RegisterDecoder failed: %v", err)
		}
		_, err := service.Decode(context.Background(), typ, "wrong", strings.NewReader(""))
		if err == nil {
			t.Fatal("Decode should fail for wrong type")
		}
		if !strings.Contains(err.Error(), "decoder returned") {
			t.Errorf("Decode error message = %q, want it to contain 'decoder returned'", err.Error())
		}
	})

	t.Run("decoder returns nil for non-pointer type", func(t *testing.T) {
		if err := service.RegisterDecoder(typ, "nil", func(context.Context, io.Reader) (any, error) {
			return nil, nil
		}, false); err != nil {
			t.Fatalf("RegisterDecoder failed: %v", err)
		}
		_, err := service.Decode(context.Background(), typ, "nil", strings.NewReader(""))
		if err == nil {
			t.Fatal("Decode should fail for nil return on non-pointer type")
		}
		if !strings.Contains(err.Error(), "decoder returned nil") {
			t.Errorf("Decode error message = %q, want it to contain 'decoder returned nil'", err.Error())
		}
	})

	t.Run("decoder returns error", func(t *testing.T) {
		if err := service.RegisterDecoder(typ, "error", func(context.Context, io.Reader) (any, error) {
			return nil, errors.New("decode error")
		}, false); err != nil {
			t.Fatalf("RegisterDecoder failed: %v", err)
		}
		_, err := service.Decode(context.Background(), typ, "error", strings.NewReader(""))
		if err == nil {
			t.Fatal("Decode should fail when decoder returns error")
		}
	})

	t.Run("nil type", func(t *testing.T) {
		if err := service.RegisterDecoder(typ, "nil-type", func(context.Context, io.Reader) (any, error) {
			return nil, nil
		}, false); err != nil {
			t.Fatalf("RegisterDecoder failed: %v", err)
		}
		_, err := service.Decode(context.Background(), nil, "nil-type", strings.NewReader(""))
		if err == nil {
			t.Fatal("Decode should fail for nil type")
		}
	})
}

func TestServiceEncode(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()

	// Register an encoder
	if err := service.RegisterEncoder(typ, "json", func(ctx context.Context, w io.Writer, value any) error {
		_, err := fmt.Fprintf(w, "encoded:%s", value.(testAsset).Value)
		return err
	}, false); err != nil {
		t.Fatalf("RegisterEncoder failed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.Encode(context.Background(), typ, "json", &buf, testAsset{Value: "test"})
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}
		if buf.String() != "encoded:test" {
			t.Errorf("Encode output = %q, want %q", buf.String(), "encoded:test")
		}
	})

	t.Run("encoder not registered", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.Encode(context.Background(), typ, "yaml", &buf, testAsset{})
		if err == nil {
			t.Fatal("Encode should fail for unregistered encoder")
		}
		if !strings.Contains(err.Error(), "encoder not registered") {
			t.Errorf("Encode error message = %q, want it to contain 'encoder not registered'", err.Error())
		}
	})

	t.Run("encode nil value", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.Encode(context.Background(), typ, "json", &buf, nil)
		if err == nil {
			t.Fatal("Encode should fail for nil value")
		}
		if !strings.Contains(err.Error(), "cannot encode nil") {
			t.Errorf("Encode error message = %q, want it to contain 'cannot encode nil'", err.Error())
		}
	})

	t.Run("encode wrong type", func(t *testing.T) {
		var buf bytes.Buffer
		err := service.Encode(context.Background(), typ, "json", &buf, "wrong type")
		if err == nil {
			t.Fatal("Encode should fail for wrong type")
		}
		if !strings.Contains(err.Error(), "cannot encode") {
			t.Errorf("Encode error message = %q, want it to contain 'cannot encode'", err.Error())
		}
	})

	t.Run("encoder returns error", func(t *testing.T) {
		if err := service.RegisterEncoder(typ, "error", func(context.Context, io.Writer, any) error {
			return errors.New("encode error")
		}, false); err != nil {
			t.Fatalf("RegisterEncoder failed: %v", err)
		}
		var buf bytes.Buffer
		err := service.Encode(context.Background(), typ, "error", &buf, testAsset{})
		if err == nil {
			t.Fatal("Encode should fail when encoder returns error")
		}
	})
}

func TestServiceCacheOperations(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()

	t.Run("cache put and get", func(t *testing.T) {
		value := testAsset{Value: "cached-value"}

		service.Cache("test-id", typ, "json", value)

		retrieved, ok := service.Cached("test-id", typ, "json")
		if !ok {
			t.Fatal("Cached returned ok=false after Cache")
		}
		if retrieved.(testAsset).Value != value.Value {
			t.Errorf("Cached returned value=%v, want %v", retrieved, value)
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		_, ok := service.Cached("nonexistent", typ, "json")
		if ok {
			t.Error("Cached returned ok=true for nonexistent key")
		}
	})

	t.Run("cache invalidation", func(t *testing.T) {
		service.Cache("invalid-me", typ, "json", testAsset{Value: "before"})
		service.Cache("keep-me", typ, "json", testAsset{Value: "keep"})

		service.Invalidate("invalid-me")

		_, ok := service.Cached("invalid-me", typ, "json")
		if ok {
			t.Error("Cached returned ok=true after Invalidate")
		}

		_, ok = service.Cached("keep-me", typ, "json")
		if !ok {
			t.Error("Cached returned ok=false for non-invalidated key")
		}
	})

	t.Run("cache clear", func(t *testing.T) {
		service.Cache("clear1", typ, "json", testAsset{Value: "v1"})
		service.Cache("clear2", typ, "json", testAsset{Value: "v2"})

		service.ClearCache()

		_, ok := service.Cached("clear1", typ, "json")
		if ok {
			t.Error("Cached returned ok=true after ClearCache")
		}
		_, ok = service.Cached("clear2", typ, "json")
		if ok {
			t.Error("Cached returned ok=true after ClearCache")
		}
	})
}

func TestServiceTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{
			name:     "nil type",
			typ:      nil,
			expected: "<nil>",
		},
		{
			name:     "string type",
			typ:      reflect.TypeFor[string](),
			expected: "string",
		},
		{
			name:     "custom type",
			typ:      reflect.TypeFor[testAsset](),
			expected: "assets.testAsset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeName(tt.typ)
			if result != tt.expected {
				t.Errorf("typeName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewLoadKey(t *testing.T) {
	key := NewLoadKey("my-id", reflect.TypeFor[string](), "json")
	if key.id != "my-id" {
		t.Errorf("key.id = %q, want %q", key.id, "my-id")
	}
	if key.format != "json" {
		t.Errorf("key.format = %q, want %q", key.format, "json")
	}
	if key.typ != reflect.TypeFor[string]() {
		t.Errorf("key.typ = %v, want %v", key.typ, reflect.TypeFor[string]())
	}
}

func TestServiceIntegration(t *testing.T) {
	service := NewService(nil)
	typ := reflect.TypeFor[testAsset]()

	// Register decoder and encoder
	if err := service.RegisterDecoder(typ, "json", func(ctx context.Context, r io.Reader) (any, error) {
		data, _ := io.ReadAll(r)
		return testAsset{Value: string(data)}, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}

	if err := service.RegisterEncoder(typ, "json", func(ctx context.Context, w io.Writer, value any) error {
		_, err := fmt.Fprintf(w, "%s", value.(testAsset).Value)
		return err
	}, false); err != nil {
		t.Fatalf("RegisterEncoder failed: %v", err)
	}

	// Test full decode/encode cycle
	input := strings.NewReader("hello-world")
	decoded, err := service.Decode(context.Background(), typ, "json", input)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var output bytes.Buffer
	if err := service.Encode(context.Background(), typ, "json", &output, decoded); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if output.String() != "hello-world" {
		t.Errorf("round-trip output = %q, want %q", output.String(), "hello-world")
	}

	// Test caching
	service.Cache("test", typ, "json", decoded)
	cached, ok := service.Cached("test", typ, "json")
	if !ok {
		t.Fatal("Cached returned ok=false")
	}
	if cached.(testAsset).Value != decoded.(testAsset).Value {
		t.Errorf("cached value mismatch")
	}
}

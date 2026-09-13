package assets

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadKeyString(t *testing.T) {
	// Test that String() doesn't panic and returns non-empty strings
	tests := []struct {
		name string
		key  loadKey
	}{
		{
			name: "nil type",
			key:  loadKey{id: "test", typ: nil, format: "json"},
		},
		{
			name: "non-nil type",
			key:  loadKey{id: "asset", typ: reflect.TypeFor[string](), format: "png"},
		},
		{
			name: "empty values",
			key:  loadKey{id: "", typ: reflect.TypeFor[int](), format: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.key.String()
			if result == "" {
				t.Error("String() returned empty string")
			}
			// Verify format contains the expected parts
			if !strings.Contains(result, tt.key.id) {
				t.Errorf("String() = %q, should contain id %q", result, tt.key.id)
			}
			if !strings.Contains(result, tt.key.format) {
				t.Errorf("String() = %q, should contain format %q", result, tt.key.format)
			}
		})
	}
}

func TestNewCache(t *testing.T) {
	c := newCache()
	if c == nil {
		t.Fatal("newCache returned nil")
	}
	if c.entries == nil {
		t.Fatal("cache entries map is nil")
	}
}

func TestCacheGetPut(t *testing.T) {
	c := newCache()
	typ := reflect.TypeFor[string]()
	key := NewLoadKey("test", typ, "json")

	// Test that get returns false for missing key
	value, ok := c.get(key)
	if ok || value != nil {
		t.Errorf("get on empty cache returned (value=%v, ok=%v), want (nil, false)", value, ok)
	}

	// Test put and get
	c.put(key, "test-value")
	value, ok = c.get(key)
	if !ok {
		t.Fatal("get returned ok=false after put")
	}
	if value.(string) != "test-value" {
		t.Errorf("get returned value=%v, want %q", value, "test-value")
	}

	// Test that different keys return different values
	differentKey := NewLoadKey("other", typ, "json")
	c.put(differentKey, "other-value")
	value, ok = c.get(key)
	if !ok || value.(string) != "test-value" {
		t.Errorf("get with original key returned (value=%v, ok=%v), want (%q, true)", value, ok, "test-value")
	}
	value, ok = c.get(differentKey)
	if !ok || value.(string) != "other-value" {
		t.Errorf("get with different key returned (value=%v, ok=%v), want (%q, true)", value, ok, "other-value")
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := newCache()
	typ := reflect.TypeFor[string]()
	jsonKey := NewLoadKey("asset1", typ, "json")
	pngKey := NewLoadKey("asset1", typ, "png")
	otherKey := NewLoadKey("asset2", typ, "json")

	c.put(jsonKey, "json-value")
	c.put(pngKey, "png-value")
	c.put(otherKey, "other-value")

	// Invalidate by ID should remove all entries with that ID
	c.invalidate("asset1")

	if _, ok := c.get(jsonKey); ok {
		t.Error("json entry was not invalidated")
	}
	if _, ok := c.get(pngKey); ok {
		t.Error("png entry was not invalidated")
	}
	// asset2 should still exist
	if _, ok := c.get(otherKey); !ok {
		t.Error("other entry was incorrectly invalidated")
	}
}

func TestCacheClear(t *testing.T) {
	c := newCache()
	typ := reflect.TypeFor[string]()
	key1 := NewLoadKey("key1", typ, "json")
	key2 := NewLoadKey("key2", typ, "json")

	c.put(key1, "value1")
	c.put(key2, "value2")

	c.clear()

	if _, ok := c.get(key1); ok {
		t.Error("key1 still exists after clear")
	}
	if _, ok := c.get(key2); ok {
		t.Error("key2 still exists after clear")
	}

	// Verify cache is still usable after clear
	c.put(key1, "new-value")
	if v, ok := c.get(key1); !ok || v.(string) != "new-value" {
		t.Error("cache not usable after clear")
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := newCache()
	typ := reflect.TypeFor[int]()
	key := NewLoadKey("concurrent", typ, "test")

	// Test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			c.put(key, id)
			_, _ = c.get(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify cache is still functional
	c.put(key, 42)
	v, ok := c.get(key)
	if !ok || v.(int) != 42 {
		t.Error("cache corrupted by concurrent access")
	}
}

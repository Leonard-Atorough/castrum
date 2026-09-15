package assets

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

type testSimpleComponent struct {
	Name  string
	Value int
}

func (s *testSimpleComponent) Serialize() map[string]any {
	return map[string]any{"Name": s.Name, "Value": s.Value}
}

func (s *testSimpleComponent) Deserialize(props map[string]any) error {
	if name, ok := props["Name"].(string); ok {
		s.Name = name
	}
	if value, ok := props["Value"].(float64); ok {
		s.Value = int(value)
	}
	return nil
}

type testAnotherComponent struct {
	Enabled bool
}

func TestNewAssets(t *testing.T) {
	assets := NewAssets(nil)
	if assets == nil {
		t.Fatal("NewAssets returned nil")
	}
	if assets.loader == nil {
		t.Fatal("Assets.loader is nil")
	}
	if assets.saver == nil {
		t.Fatal("Assets.saver is nil")
	}
}

func TestAssetsAccessors(t *testing.T) {
	assets := NewAssets(nil)

	t.Run("AssetLoader", func(t *testing.T) {
		loader := assets.AssetLoader()
		if loader == nil {
			t.Fatal("AssetLoader() returned nil")
		}
	})

	t.Run("AssetSaver", func(t *testing.T) {
		saver := assets.AssetSaver()
		if saver == nil {
			t.Fatal("AssetSaver() returned nil")
		}
	})
}

func TestBlueprintYAML(t *testing.T) {
	t.Run("decode blueprint YAML", func(t *testing.T) {
		yamlInput := `
name: TestEntity
version: "1.0"
components:
  - type: test
    properties:
      key: value
`
		blueprint, err := decodeBlueprintYAML(context.Background(), strings.NewReader(yamlInput))
		if err != nil {
			t.Fatalf("decodeBlueprintYAML failed: %v", err)
		}
		if blueprint.Name != "TestEntity" {
			t.Errorf("Name = %q, want %q", blueprint.Name, "TestEntity")
		}
		if blueprint.Version != "1.0" {
			t.Errorf("Version = %q, want %q", blueprint.Version, "1.0")
		}
		if len(blueprint.Components) != 1 {
			t.Errorf("Components length = %d, want 1", len(blueprint.Components))
		}
		if blueprint.Components[0].Type != "test" {
			t.Errorf("Component Type = %q, want %q", blueprint.Components[0].Type, "test")
		}
	})

	t.Run("encode blueprint YAML", func(t *testing.T) {
		blueprint := Blueprint{
			Name:    "TestEntity",
			Version: "1.0",
			Components: []ComponentData{
				{Type: "test", Properties: map[string]any{"key": "value"}},
			},
		}
		var buf bytes.Buffer
		err := encodeBlueprintYAML(context.Background(), &buf, blueprint)
		if err != nil {
			t.Fatalf("encodeBlueprintYAML failed: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "TestEntity") {
			t.Errorf("encoded output doesn't contain entity name")
		}
		if !strings.Contains(output, "test") {
			t.Errorf("encoded output doesn't contain component type")
		}
	})

	t.Run("decode blueprint YAML error", func(t *testing.T) {
		_, err := decodeBlueprintYAML(context.Background(), strings.NewReader("invalid: yaml: [[["))
		if err == nil {
			t.Fatal("decodeBlueprintYAML should fail with invalid YAML")
		}
	})
}

func TestTextureCodec(t *testing.T) {
	// Create a simple 2x2 image
	rect := image.Rect(0, 0, 2, 2)
	img := image.NewRGBA(rect)
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(0, 1, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	img.Set(1, 0, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	t.Run("decode texture PNG", func(t *testing.T) {
		// Create a simple PNG
		var buf bytes.Buffer
		if err := encodeTexturePNG(context.Background(), &buf, TextureData{Image: img, Width: 2, Height: 2}); err != nil {
			t.Fatalf("encodeTexturePNG failed: %v", err)
		}

		// Decode it back
		decoded, err := decodeTexture(context.Background(), &buf)
		if err != nil {
			t.Fatalf("decodeTexture failed: %v", err)
		}
		if decoded.Width != 2 || decoded.Height != 2 {
			t.Errorf("decoded dimensions = (%d, %d), want (2, 2)", decoded.Width, decoded.Height)
		}
		if decoded.Image == nil {
			t.Fatal("decoded image is nil")
		}
	})

	t.Run("encode texture PNG with nil image", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeTexturePNG(context.Background(), &buf, TextureData{Image: nil})
		if err == nil {
			t.Fatal("encodeTexturePNG should fail with nil image")
		}
		if !strings.Contains(err.Error(), "nil") {
			t.Errorf("error message should mention nil, got: %v", err)
		}
	})

	t.Run("encode texture JPEG with nil image", func(t *testing.T) {
		var buf bytes.Buffer
		err := encodeTextureJPEG(context.Background(), &buf, TextureData{Image: nil})
		if err == nil {
			t.Fatal("encodeTextureJPEG should fail with nil image")
		}
		if !strings.Contains(err.Error(), "nil") {
			t.Errorf("error message should mention nil, got: %v", err)
		}
	})
}

func TestCreateFromBlueprint(t *testing.T) {
	// Use explicit factories for better control
	registry := NewComponentRegistry()

	// Factory that properly handles properties
	factory := func(props map[string]any) (ecs.Component, error) {
		name := ""
		value := 0
		if n, ok := props["Name"].(string); ok {
			name = n
		}
		if v, ok := props["Value"].(float64); ok {
			value = int(v)
		} else if v, ok := props["Value"].(int); ok {
			value = v
		}
		return &testSimpleComponent{Name: name, Value: value}, nil
	}

	if err := registry.RegisterComponent("testComponent", factory); err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "TestEntity",
			Components: []ComponentData{
				{Type: "testComponent", Properties: map[string]any{"Name": "custom", "Value": 100}},
			},
		}

		entity, err := CreateFromBlueprint(world, registry, bp)
		if err != nil {
			t.Fatalf("CreateFromBlueprint failed: %v", err)
		}
		if entity == nil {
			t.Fatal("CreateFromBlueprint returned nil entity")
		}

		comp, err := world.GetComponent[*testSimpleComponent](entity.ID)
		if err != nil {
			t.Fatalf("GetComponent failed: %v", err)
		}
		if comp.Name != "custom" {
			t.Errorf("Name = %q, want %q", comp.Name, "custom")
		}
		if comp.Value != 100 {
			t.Errorf("Value = %d, want 100", comp.Value)
		}
	})

	t.Run("unregistered component", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "BrokenEntity",
			Components: []ComponentData{
				{Type: "unregisteredComponent", Properties: nil},
			},
		}

		_, err := CreateFromBlueprint(world, registry, bp)
		if err == nil {
			t.Fatal("CreateFromBlueprint should fail with unregistered component")
		}
	})

	t.Run("multiple components", func(t *testing.T) {
		// Register another component type using Register instead of RegisterComponent
		// to specify the exact type
		if err := registry.Register("testAnotherComponent", reflect.TypeFor[*testAnotherComponent](), func(props map[string]any) (ecs.Component, error) {
			return &testAnotherComponent{Enabled: true}, nil
		}); err != nil {
			t.Fatalf("Register for testAnotherComponent failed: %v", err)
		}

		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "MultiEntity",
			Components: []ComponentData{
				{Type: "testComponent", Properties: map[string]any{"Name": "multi", "Value": 200}},
				{Type: "testAnotherComponent", Properties: nil},
			},
		}

		entity, err := CreateFromBlueprint(world, registry, bp)
		if err != nil {
			t.Fatalf("CreateFromBlueprint failed: %v", err)
		}

		comp, err := world.GetComponent[*testSimpleComponent](entity.ID)
		if err != nil {
			t.Fatalf("GetComponent for testComponent failed: %v", err)
		}
		if comp.Name != "multi" || comp.Value != 200 {
			t.Errorf("testComponent not set correctly: Name=%q, Value=%d", comp.Name, comp.Value)
		}

		comp2, err := world.GetComponent[*testAnotherComponent](entity.ID)
		if err != nil {
			t.Fatalf("GetComponent for testAnotherComponent failed: %v", err)
		}
		if !comp2.Enabled {
			t.Error("testAnotherComponent.Enabled should be true")
		}
	})
}

func TestAssetError(t *testing.T) {
	t.Run("Error with nil inner error", func(t *testing.T) {
		err := &AssetError{
			Message: "test error",
			Err:     nil,
			Source:  "TestSource",
		}
		expected := "test error (source: TestSource)"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Error with inner error", func(t *testing.T) {
		innerErr := errors.New("inner error")
		err := &AssetError{
			Message: "wrapper error",
			Err:     innerErr,
			Source:  "TestSource",
		}
		expected := "wrapper error: inner error (source: TestSource)"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		innerErr := errors.New("inner error")
		err := &AssetError{
			Message: "wrapper",
			Err:     innerErr,
			Source:  "Source",
		}
		if err.Unwrap() != innerErr {
			t.Error("Unwrap() should return inner error")
		}
	})

	t.Run("Unwrap with nil", func(t *testing.T) {
		err := &AssetError{
			Message: "no inner error",
			Err:     nil,
			Source:  "Source",
		}
		if err.Unwrap() != nil {
			t.Error("Unwrap() should return nil when Err is nil")
		}
	})
}

func TestResolveFormat(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		explicit Format
		want     Format
	}{
		{
			name:     "explicit format overrides extension",
			path:     "test.png",
			explicit: FormatYAML,
			want:     FormatYAML,
		},
		{
			name:     "yaml extension",
			path:     "test.yaml",
			explicit: "",
			want:     FormatYAML,
		},
		{
			name:     "yml extension",
			path:     "test.yml",
			explicit: "",
			want:     FormatYML,
		},
		{
			name:     "jpeg extension",
			path:     "test.jpeg",
			explicit: "",
			want:     FormatJPEG,
		},
		{
			name:     "jpg extension",
			path:     "test.jpg",
			explicit: "",
			want:     Format("jpg"),
		},
		{
			name:     "png extension",
			path:     "test.png",
			explicit: "",
			want:     FormatPNG,
		},
		{
			name:     "unknown extension",
			path:     "test.unknown",
			explicit: "",
			want:     Format("unknown"),
		},
		{
			name:     "no extension",
			path:     "test",
			explicit: "",
			want:     Format(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFormat(tt.path, tt.explicit)
			if got != tt.want {
				t.Errorf("resolveFormat(%q, %q) = %q, want %q", tt.path, tt.explicit, got, tt.want)
			}
		})
	}
}

func TestNormalizePaths(t *testing.T) {
	t.Run("normalizeAssetPath", func(t *testing.T) {
		tests := []struct {
			path string
			want string
		}{
			{"/path/to/asset", "/path/to/asset"},
			{"/path/../asset", "/asset"},
			{"./asset", "asset"},
			{"/path//to/asset", "/path/to/asset"},
		}
		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				got := normalizeAssetPath(tt.path)
				if got != tt.want {
					t.Errorf("normalizeAssetPath(%q) = %q, want %q", tt.path, got, tt.want)
				}
			})
		}
	})

	t.Run("normalizeSavePath", func(t *testing.T) {
		// normalizeSavePath uses filepath.Clean which is OS-specific
		// We'll just verify it doesn't panic and returns a non-empty string for valid inputs
		tests := []string{"/path/to/asset", "/path/../asset", "./asset"}
		for _, path := range tests {
			t.Run(path, func(t *testing.T) {
				got := normalizeSavePath(path)
				if got == "" {
					t.Errorf("normalizeSavePath(%q) returned empty string", path)
				}
			})
		}
	})

	t.Run("assetIDForSavePath", func(t *testing.T) {
		tests := []struct {
			path string
			want string
		}{
			{"/path/to/asset", "/path/to/asset"},
			{"/path/../asset", "/asset"},
		}
		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				got := assetIDForSavePath(tt.path)
				if string(got) != tt.want {
					t.Errorf("assetIDForSavePath(%q) = %q, want %q", tt.path, string(got), tt.want)
				}
			})
		}
		// Backslash-to-slash conversion is only meaningful on Windows.
		if runtime.GOOS == "windows" {
			got := assetIDForSavePath("path\\to\\asset")
			if string(got) != "path/to/asset" {
				t.Errorf("assetIDForSavePath(%q) = %q, want %q", "path\\to\\asset", string(got), "path/to/asset")
			}
		}
	})
}

func TestResolveLoadOptions(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		opts := resolveLoadOptions("test.png")
		if opts.ID != ID("test.png") {
			t.Errorf("ID = %q, want %q", opts.ID, ID("test.png"))
		}
		if opts.Format != FormatPNG {
			t.Errorf("Format = %q, want %q", opts.Format, FormatPNG)
		}
		if opts.CachePolicy != CachePolicyDefault {
			t.Errorf("CachePolicy = %d, want %d", opts.CachePolicy, CachePolicyDefault)
		}
	})

	t.Run("with options", func(t *testing.T) {
		opts := resolveLoadOptions("test.png", WithID("custom-id"), WithFormat(FormatYAML), WithCache(CachePolicyNone))
		if opts.ID != ID("custom-id") {
			t.Errorf("ID = %q, want %q", opts.ID, ID("custom-id"))
		}
		if opts.Format != FormatYAML {
			t.Errorf("Format = %q, want %q", opts.Format, FormatYAML)
		}
		if opts.CachePolicy != CachePolicyNone {
			t.Errorf("CachePolicy = %d, want %d", opts.CachePolicy, CachePolicyNone)
		}
	})
}

func TestNewAssetService(t *testing.T) {
	service := newAssetService(nil)
	if service == nil {
		t.Fatal("newAssetService returned nil")
	}
}

func TestRegisterDefaultDecoders(t *testing.T) {
	service := newAssetService(nil)
	if service == nil {
		t.Fatal("service is nil")
	}
}

func TestRegisterDefaultEncoder(t *testing.T) {
	service := newAssetService(nil)
	if service == nil {
		t.Fatal("service is nil")
	}
}

func TestTextureData(t *testing.T) {
	rect := image.Rect(0, 0, 10, 10)
	img := image.NewRGBA(rect)

	texture := TextureData{
		Image:  img,
		Width:  10,
		Height: 10,
	}

	if texture.Width != 10 {
		t.Errorf("Width = %d, want 10", texture.Width)
	}
	if texture.Height != 10 {
		t.Errorf("Height = %d, want 10", texture.Height)
	}
	if texture.Image == nil {
		t.Error("Image is nil")
	}
}

func TestFormatConstants(t *testing.T) {
	if FormatJSON != "json" {
		t.Errorf("FormatJSON = %q, want %q", FormatJSON, "json")
	}
	if FormatPNG != "png" {
		t.Errorf("FormatPNG = %q, want %q", FormatPNG, "png")
	}
	if FormatJPEG != "jpeg" {
		t.Errorf("FormatJPEG = %q, want %q", FormatJPEG, "jpeg")
	}
	if FormatYAML != "yaml" {
		t.Errorf("FormatYAML = %q, want %q", FormatYAML, "yaml")
	}
	if FormatYML != "yml" {
		t.Errorf("FormatYML = %q, want %q", FormatYML, "yml")
	}
}

func TestBlueprint(t *testing.T) {
	bp := Blueprint{
		Name:    "Test",
		Version: "1.0",
		Components: []ComponentData{
			{Type: "comp1", Properties: map[string]any{"key": "value"}},
		},
	}

	if bp.Name != "Test" {
		t.Errorf("Name = %q, want %q", bp.Name, "Test")
	}
	if bp.Version != "1.0" {
		t.Errorf("Version = %q, want %q", bp.Version, "1.0")
	}
	if len(bp.Components) != 1 {
		t.Errorf("len(Components) = %d, want 1", len(bp.Components))
	}
}

func TestComponentData(t *testing.T) {
	cd := ComponentData{
		Type:       "test",
		Properties: map[string]any{"int": 42, "string": "value"},
	}

	if cd.Type != "test" {
		t.Errorf("Type = %q, want %q", cd.Type, "test")
	}
	if cd.Properties["int"] != 42 {
		t.Errorf("Properties[\"int\"] = %v, want 42", cd.Properties["int"])
	}
	if cd.Properties["string"] != "value" {
		t.Errorf("Properties[\"string\"] = %v, want %q", cd.Properties["string"], "value")
	}
}

func TestIDAndFormatTypes(t *testing.T) {
	var id ID = "test-id"
	var format Format = "test-format"

	if string(id) != "test-id" {
		t.Errorf("ID conversion = %q, want %q", string(id), "test-id")
	}
	if string(format) != "test-format" {
		t.Errorf("Format conversion = %q, want %q", string(format), "test-format")
	}
}

package assets

import (
	"testing"
	"testing/fstest"
)

func TestNewAssets(t *testing.T) {
	a := NewAssets(nil)
	if a.Textures == nil || a.Blueprints == nil {
		t.Fatal("expected both stores to be initialized")
	}
}

func TestAssetsLoad(t *testing.T) {
	fs := fstest.MapFS{
		"test.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	tests := []struct {
		name      string
		path      string
		wantNil   bool
		wantError bool
	}{
		{"blueprint .yaml", "test.yaml", false, false},
		{"unknown extension", "test.txt", true, false},
		{"no extension", "noext", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := a.Load(tt.path)
			if (err != nil) != tt.wantError {
				t.Fatalf("Load(%q): got err %v, wantError %v", tt.path, err, tt.wantError)
			}
			if (res == nil) != tt.wantNil {
				t.Fatalf("Load(%q): got nil %v, wantNil %v", tt.path, res == nil, tt.wantNil)
			}
		})
	}
}

func TestHasTextureExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"img.png", true},
		{"photo.jpg", true},
		{"pic.jpeg", true},
		{"file.txt", false},
		{"no_ext", false},
		{".png", true},
		{"", false},
	}

	for _, tt := range tests {
		if got := hasTextureExtension(tt.path); got != tt.want {
			t.Errorf("hasTextureExtension(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestHasBlueprintExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"blueprint.yaml", true},
		{"scene.yaml", true},
		{".yaml", true},
		{"file.txt", false},
		{"file.yml", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := hasBlueprintExtension(tt.path); got != tt.want {
			t.Errorf("hasBlueprintExtension(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

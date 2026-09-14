package atlas

import (
	"fmt"
	"testing"
)

func TestAtlasRegionBounds(t *testing.T) {
	tests := []struct {
		name     string
		region   AtlasRegion
		wantMinX float64
		wantMinY float64
		wantMaxX float64
		wantMaxY float64
	}{
		{
			name:     "region at origin",
			region:   AtlasRegion{Name: "test", X: 0, Y: 0, W: 32, H: 32},
			wantMinX: 0,
			wantMinY: 0,
			wantMaxX: 32,
			wantMaxY: 32,
		},
		{
			name:     "region at offset",
			region:   AtlasRegion{Name: "test", X: 10, Y: 20, W: 16, H: 16},
			wantMinX: 10,
			wantMinY: 20,
			wantMaxX: 26,
			wantMaxY: 36,
		},
		{
			name:     "zero-sized region",
			region:   AtlasRegion{Name: "test", X: 5, Y: 5, W: 0, H: 0},
			wantMinX: 5,
			wantMinY: 5,
			wantMaxX: 5,
			wantMaxY: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounds := tt.region.Bounds()
			if bounds.Min.X != tt.wantMinX {
				t.Errorf("Bounds().Min.X = %v, want %v", bounds.Min.X, tt.wantMinX)
			}
			if bounds.Min.Y != tt.wantMinY {
				t.Errorf("Bounds().Min.Y = %v, want %v", bounds.Min.Y, tt.wantMinY)
			}
			if bounds.Max.X != tt.wantMaxX {
				t.Errorf("Bounds().Max.X = %v, want %v", bounds.Max.X, tt.wantMaxX)
			}
			if bounds.Max.Y != tt.wantMaxY {
				t.Errorf("Bounds().Max.Y = %v, want %v", bounds.Max.Y, tt.wantMaxY)
			}
		})
	}
}

func TestNewTextureAtlas(t *testing.T) {
	atlas := NewTextureAtlas("test", "texture.png", 256, 128, make(map[string]AtlasRegion))

	if atlas.ID() != ID("test") {
		t.Errorf("ID() = %v, want %v", atlas.ID(), ID("test"))
	}
	if atlas.AssetID() != "texture.png" {
		t.Errorf("AssetID() = %v, want %v", atlas.AssetID(), "texture.png")
	}
	w, h := atlas.Dimensions()
	if w != 256 || h != 128 {
		t.Errorf("Dimensions() = (%v, %v), want (256, 128)", w, h)
	}
}

func TestAtlasRegionAccess(t *testing.T) {
	regions := map[string]AtlasRegion{
		"idle": {Name: "idle", X: 0, Y: 0, W: 32, H: 32},
		"run":  {Name: "run", X: 32, Y: 0, W: 32, H: 32},
		"jump": {Name: "jump", X: 64, Y: 0, W: 32, H: 32},
	}

	atlas := NewTextureAtlas("player", "player.png", 128, 32, regions)

	region, ok := atlas.Region("idle")
	if !ok {
		t.Fatal("Region(\"idle\") not found")
	}
	if region.X != 0 || region.Y != 0 || region.W != 32 || region.H != 32 {
		t.Errorf("Region(\"idle\") = (%d,%d,%d,%d), want (0,0,32,32)", region.X, region.Y, region.W, region.H)
	}

	_, ok = atlas.Region("nonexistent")
	if ok {
		t.Error("Region(\"nonexistent\") found, want not found")
	}

	returnedRegions := atlas.Regions()
	if len(returnedRegions) != 3 {
		t.Errorf("Regions() returned %d regions, want 3", len(returnedRegions))
	}

	returnedRegions["new"] = AtlasRegion{Name: "new", X: 0, Y: 0, W: 1, H: 1}
	if _, ok := atlas.Region("new"); ok {
		t.Error("Regions() returned map that can be modified to affect original")
	}
}

func TestAtlasError(t *testing.T) {
	err := &AtlasError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("AtlasError.Error() = %q, want %q", err.Error(), "test error")
	}
}

func TestAtlasErrorList(t *testing.T) {
	errs := AtlasErrorList{
		Errors: []error{
			fmt.Errorf("error1"),
			fmt.Errorf("error2"),
		},
	}

	errStr := errs.Error()
	if errStr == "" {
		t.Error("AtlasErrorList.Error() returned empty string")
	}
	if !contains(errStr, "error1") || !contains(errStr, "error2") {
		t.Errorf("AtlasErrorList.Error() = %q, want to contain both errors", errStr)
	}
}

func contains(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

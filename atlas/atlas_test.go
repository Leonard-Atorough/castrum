package atlas

import (
	"fmt"
	"testing"

	"github.com/leonard-atorough/castrum/assets"
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

	// Test Region
	region, ok := atlas.Region("idle")
	if !ok {
		t.Fatal("Region(\"idle\") not found")
	}
	if region.X != 0 || region.Y != 0 || region.W != 32 || region.H != 32 {
		t.Errorf("Region(\"idle\") = (%d,%d,%d,%d), want (0,0,32,32)", region.X, region.Y, region.W, region.H)
	}

	// Test Region not found
	_, ok = atlas.Region("nonexistent")
	if ok {
		t.Error("Region(\"nonexistent\") found, want not found")
	}

	// Test Regions() returns copy
	returnedRegions := atlas.Regions()
	if len(returnedRegions) != 3 {
		t.Errorf("Regions() returned %d regions, want 3", len(returnedRegions))
	}

	// Modify returned map should not affect original
	returnedRegions["new"] = AtlasRegion{Name: "new", X: 0, Y: 0, W: 1, H: 1}
	if _, ok := atlas.Region("new"); ok {
		t.Error("Regions() returned map that can be modified to affect original")
	}
}

func TestBuilderSliceRegion(t *testing.T) {
	store := &mockAtlasStorer{}
	builder := NewBuilder("test", "texture.png", 256, 128, store)

	// Test successful slice
	b, err := builder.SliceRegion("idle", 0, 0, 32, 32)
	if err != nil {
		t.Errorf("SliceRegion() error = %v, want nil", err)
	}
	if b == nil {
		t.Fatal("SliceRegion() returned nil builder")
	}

	// Test duplicate region
	b, err = b.SliceRegion("idle", 10, 10, 32, 32)
	if err == nil {
		t.Error("SliceRegion() with duplicate name error = nil, want error")
	}

	// Test out of bounds (x+w > texW)
	b, err = b.SliceRegion("oob", 300, 0, 32, 32)
	if err == nil {
		t.Error("SliceRegion() out of bounds (x+w) error = nil, want error")
	}

	// Test out of bounds (y+h > texH)
	b, err = b.SliceRegion("oob2", 0, 200, 32, 32)
	if err == nil {
		t.Error("SliceRegion() out of bounds (y+h) error = nil, want error")
	}

	// Test negative width
	b, err = b.SliceRegion("negative_w", 0, 0, -1, 32)
	if err == nil {
		t.Error("SliceRegion() negative width error = nil, want error")
	}

	// Test negative height
	b, err = b.SliceRegion("negative_h", 0, 0, 32, -1)
	if err == nil {
		t.Error("SliceRegion() negative height error = nil, want error")
	}

	// Test overlapping regions
	b, err = b.SliceRegion("overlap", 16, 16, 32, 32)
	if err == nil {
		t.Error("SliceRegion() overlapping error = nil, want error")
	}
}

func TestBuilderGridSlice(t *testing.T) {
	store := &mockAtlasStorer{}
	builder := NewBuilder("grid", "grid.png", 64, 64, store)

	// 4x4 grid of 16x16 frames
	b, errs := builder.GridSlice(16, 16, func(idx int) string {
		return fmt.Sprintf("frame_%d", idx)
	})

	if len(errs) > 0 {
		t.Errorf("GridSlice() errors = %v, want nil", errs)
	}

	// Build and verify
	atlas, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have 16 regions (4x4)
	if len(atlas.Regions()) != 16 {
		t.Errorf("GridSlice() created %d regions, want 16", len(atlas.Regions()))
	}

	// Verify frame_0
	region, ok := atlas.Region("frame_0")
	if !ok {
		t.Fatal("frame_0 not found")
	}
	if region.X != 0 || region.Y != 0 || region.W != 16 || region.H != 16 {
		t.Errorf("frame_0 = (%d,%d,%d,%d), want (0,0,16,16)", region.X, region.Y, region.W, region.H)
	}

	// Verify frame_15 (bottom-right)
	region, ok = atlas.Region("frame_15")
	if !ok {
		t.Fatal("frame_15 not found")
	}
	if region.X != 48 || region.Y != 48 {
		t.Errorf("frame_15 position = (%d,%d), want (48,48)", region.X, region.Y)
	}
}

func TestBuilderFromMeta(t *testing.T) {
	store := &mockAtlasStorer{}
	builder := NewBuilder("meta", "meta.png", 256, 256, store)

	meta := assets.AtlasMeta{
		Regions: []assets.AtlasRegionMeta{
			{Name: "region1", X: 0, Y: 0, W: 64, H: 64},
			{Name: "region2", X: 64, Y: 0, W: 64, H: 64},
			{Name: "region3", X: 0, Y: 64, W: 64, H: 64},
		},
	}

	b, err := builder.FromMeta(meta)
	if err != nil {
		t.Fatalf("FromMeta() error = %v", err)
	}

	atlas, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(atlas.Regions()) != 3 {
		t.Errorf("FromMeta() created %d regions, want 3", len(atlas.Regions()))
	}

	region, ok := atlas.Region("region2")
	if !ok {
		t.Fatal("region2 not found")
	}
	if region.X != 64 || region.Y != 0 || region.W != 64 || region.H != 64 {
		t.Errorf("region2 = (%d,%d,%d,%d), want (64,0,64,64)", region.X, region.Y, region.W, region.H)
	}
}

func TestBuilderBuild(t *testing.T) {
	store := &mockAtlasStorer{}
	builder := NewBuilder("build_test", "build.png", 1024, 512, store)

	// Empty builder currently succeeds (no validation in Build)
	atlas, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() with no regions error = %v, want nil", err)
	}
	if len(atlas.Regions()) != 0 {
		t.Errorf("Build() with no regions returned atlas with %d regions, want 0", len(atlas.Regions()))
	}

	// Add a region and build
	builder2 := NewBuilder("build_test2", "build.png", 1024, 512, store)
	b, _ := builder2.SliceRegion("valid", 0, 0, 100, 100)
	atlas, err = b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if atlas.ID() != ID("build_test2") {
		t.Errorf("Build() atlas ID = %v, want %v", atlas.ID(), ID("build_test2"))
	}
	if atlas.AssetID() != "build.png" {
		t.Errorf("Build() atlas AssetID = %v, want %v", atlas.AssetID(), "build.png")
	}

	// Verify store was called
	if store.lastAtlasID != "build_test2" {
		t.Errorf("store.Set() was not called with correct atlasID, got %q", store.lastAtlasID)
	}
	if store.lastAssetID != "build.png" {
		t.Errorf("store.Set() was not called with correct assetID, got %q", store.lastAssetID)
	}
	if store.lastAtlas == nil {
		t.Error("store.Set() was not called with atlas")
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
	// Verify it contains both error messages
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

type mockAtlasStorer struct {
	lastAtlasID string
	lastAssetID string
	lastAtlas   any
}

func (m *mockAtlasStorer) Set(atlasID, assetID string, atlas any) error {
	m.lastAtlasID = atlasID
	m.lastAssetID = assetID
	m.lastAtlas = atlas
	return nil
}

package atlas

import (
	"testing"

	"github.com/leonard-atorough/castrum/assets"
)

func newTestService() *Service {
	return NewService(NewStore())
}

func TestBuilderSliceRegion(t *testing.T) {
	svc := newTestService()
	builder, err := svc.NewBuilder("test", "texture.png", 256, 128)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	b, err := builder.SliceRegion("idle", 0, 0, 32, 32)
	if err != nil {
		t.Errorf("SliceRegion() error = %v, want nil", err)
	}
	if b == nil {
		t.Fatal("SliceRegion() returned nil builder")
	}

	b, err = b.SliceRegion("idle", 10, 10, 32, 32)
	if err == nil {
		t.Error("SliceRegion() with duplicate name error = nil, want error")
	}

	b, err = b.SliceRegion("oob", 300, 0, 32, 32)
	if err == nil {
		t.Error("SliceRegion() out of bounds (x+w) error = nil, want error")
	}

	b, err = b.SliceRegion("oob2", 0, 200, 32, 32)
	if err == nil {
		t.Error("SliceRegion() out of bounds (y+h) error = nil, want error")
	}

	b, err = b.SliceRegion("negative_w", 0, 0, -1, 32)
	if err == nil {
		t.Error("SliceRegion() negative width error = nil, want error")
	}

	b, err = b.SliceRegion("negative_h", 0, 0, 32, -1)
	if err == nil {
		t.Error("SliceRegion() negative height error = nil, want error")
	}

	b, err = b.SliceRegion("overlap", 16, 16, 32, 32)
	if err == nil {
		t.Error("SliceRegion() overlapping error = nil, want error")
	}
}

func TestBuilderGridSlice(t *testing.T) {
	svc := newTestService()
	builder, err := svc.NewBuilder("grid", "grid.png", 64, 64)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	b, errs := builder.GridSlice(16, 16, "frame")

	if len(errs) > 0 {
		t.Errorf("GridSlice() errors = %v, want nil", errs)
	}

	atlas, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(atlas.Regions()) != 16 {
		t.Errorf("GridSlice() created %d regions, want 16", len(atlas.Regions()))
	}

	region, ok := atlas.Region("frame_0")
	if !ok {
		t.Fatal("frame_0 not found")
	}
	if region.X != 0 || region.Y != 0 || region.W != 16 || region.H != 16 {
		t.Errorf("frame_0 = (%d,%d,%d,%d), want (0,0,16,16)", region.X, region.Y, region.W, region.H)
	}

	region, ok = atlas.Region("frame_15")
	if !ok {
		t.Fatal("frame_15 not found")
	}
	if region.X != 48 || region.Y != 48 {
		t.Errorf("frame_15 position = (%d,%d), want (48,48)", region.X, region.Y)
	}
}

func TestBuilderFromMeta(t *testing.T) {
	svc := newTestService()
	builder, err := svc.NewBuilder("meta", "meta.png", 256, 256)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	meta := assets.AtlasMeta{
		Regions: []assets.AtlasRegionMeta{
			{Name: "region1", X: 0, Y: 0, W: 64, H: 64},
			{Name: "region2", X: 64, Y: 0, W: 64, H: 64},
			{Name: "region3", X: 0, Y: 64, W: 64, H: 64},
		},
	}

	b, errs := builder.FromMeta(meta)
	if len(errs) > 0 {
		t.Fatalf("FromMeta() errors = %v", errs)
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
	svc := newTestService()
	builder, err := svc.NewBuilder("build_test", "build.png", 1024, 512)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	atlas, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() with no regions error = %v, want nil", err)
	}
	if len(atlas.Regions()) != 0 {
		t.Errorf("Build() with no regions returned atlas with %d regions, want 0", len(atlas.Regions()))
	}

	builder2, err := svc.NewBuilder("build_test2", "build.png", 1024, 512)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}
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

	// Verify the atlas was registered with the service
	got, err := svc.Get("build_test2", "build.png")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil {
		t.Error("Build() did not register atlas with the service")
	}
}

func TestNewBuilderRejectsInvalidDimensions(t *testing.T) {
	svc := newTestService()
	tests := []struct {
		name    string
		texW    int
		texH    int
		wantErr bool
	}{
		{"zero width", 0, 64, true},
		{"zero height", 64, 0, true},
		{"negative width", -1, 64, true},
		{"negative height", 64, -1, true},
		{"valid dimensions", 64, 64, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.NewBuilder("test", "tex.png", tt.texW, tt.texH)
			if tt.wantErr && err == nil {
				t.Error("NewBuilder() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("NewBuilder() error = %v, want nil", err)
			}
		})
	}
}

func TestBuilderGridSliceRejectsNonDivisibleDimensions(t *testing.T) {
	svc := newTestService()
	tests := []struct {
		name  string
		texW  int
		texH  int
		tileW int
		tileH int
		errs  bool
	}{
		{"evenly divisible", 64, 64, 16, 16, false},
		{"width not divisible", 64, 64, 24, 16, true},
		{"height not divisible", 64, 64, 16, 24, true},
		{"both not divisible", 64, 64, 24, 24, true},
		{"zero tile width", 64, 64, 0, 16, true},
		{"negative tile height", 64, 64, 16, -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := svc.NewBuilder("grid", "grid.png", tt.texW, tt.texH)
			if err != nil {
				t.Fatalf("NewBuilder() error = %v", err)
			}
			_, errs := builder.GridSlice(tt.tileW, tt.tileH, "tile")
			if tt.errs && len(errs) == 0 {
				t.Error("GridSlice() errors = nil, want errors")
			}
			if !tt.errs && len(errs) > 0 {
				t.Errorf("GridSlice() errors = %v, want nil", errs)
			}
		})
	}
}

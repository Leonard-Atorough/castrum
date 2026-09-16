package atlas

import (
	"testing"

	"github.com/leonard-atorough/castrum/assets"
)

func newTestService() *Service {
	return NewService(NewStore())
}

func TestBuilderSliceRegionValid(t *testing.T) {
	svc := newTestService()
	b := svc.NewBuilder("test", "texture.png", 256, 128).
		SliceRegion("idle", 0, 0, 32, 32)
	if b == nil {
		t.Fatal("SliceRegion() returned nil builder")
	}

	atlas, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v, want nil", err)
	}
	if _, ok := atlas.Region("idle"); !ok {
		t.Error("Build() atlas missing region 'idle'")
	}
}

func TestBuilderSliceRegionErrors(t *testing.T) {
	svc := newTestService()
	tests := []struct {
		name   string
		region func(*Builder) *Builder
	}{
		{"duplicate name", func(b *Builder) *Builder {
			return b.SliceRegion("idle", 0, 0, 32, 32).
				SliceRegion("idle", 10, 10, 32, 32)
		}},
		{"out of bounds x+w", func(b *Builder) *Builder {
			return b.SliceRegion("oob", 300, 0, 32, 32)
		}},
		{"out of bounds y+h", func(b *Builder) *Builder {
			return b.SliceRegion("oob2", 0, 200, 32, 32)
		}},
		{"negative width", func(b *Builder) *Builder {
			return b.SliceRegion("negw", 0, 0, -1, 32)
		}},
		{"negative height", func(b *Builder) *Builder {
			return b.SliceRegion("negh", 0, 0, 32, -1)
		}},
		{"overlap", func(b *Builder) *Builder {
			return b.SliceRegion("idle", 0, 0, 32, 32).
				SliceRegion("overlap", 16, 16, 32, 32)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.region(svc.NewBuilder("test", "texture.png", 256, 128))
			if _, err := b.Build(); err == nil {
				t.Error("Build() error = nil, want error")
			}
		})
	}
}

func TestBuilderGridSlice(t *testing.T) {
	svc := newTestService()
	b := svc.NewBuilder("grid", "grid.png", 64, 64).
		GridSlice(16, 16, "frame")

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
	meta := assets.AtlasMeta{
		Regions: []assets.AtlasRegionMeta{
			{Name: "region1", X: 0, Y: 0, W: 64, H: 64},
			{Name: "region2", X: 64, Y: 0, W: 64, H: 64},
			{Name: "region3", X: 0, Y: 64, W: 64, H: 64},
		},
	}

	b := svc.NewBuilder("meta", "meta.png", 256, 256).
		FromMeta(meta)

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
	builder := svc.NewBuilder("build_test", "build.png", 1024, 512)

	atlas, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() with no regions error = %v, want nil", err)
	}
	if len(atlas.Regions()) != 0 {
		t.Errorf("Build() with no regions returned atlas with %d regions, want 0", len(atlas.Regions()))
	}

	builder2 := svc.NewBuilder("build_test2", "build.png", 1024, 512).
		SliceRegion("valid", 0, 0, 100, 100)
	atlas, err = builder2.Build()
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
	got, err := svc.Get("build_test2")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil {
		t.Error("Build() did not register atlas with the service")
	}
}

func TestBuildRejectsInvalidDimensions(t *testing.T) {
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
			builder := svc.NewBuilder("test", "tex.png", tt.texW, tt.texH)
			_, err := builder.Build()
			if tt.wantErr && err == nil {
				t.Error("Build() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Build() error = %v, want nil", err)
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
			builder := svc.NewBuilder("grid", "grid.png", tt.texW, tt.texH)
			_, err := builder.GridSlice(tt.tileW, tt.tileH, "tile").Build()
			if tt.errs && err == nil {
				t.Error("Build() error = nil, want errors")
			}
			if !tt.errs && err != nil {
				t.Errorf("Build() error = %v, want nil", err)
			}
		})
	}
}

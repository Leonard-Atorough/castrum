package asset

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func TestDecodeAtlasMetaRegions(t *testing.T) {
	meta, err := decodeAtlasMeta(strings.NewReader(
		`{"regions":[{"name":"player","x":1,"y":2,"w":3,"h":4}]}`))
	if err != nil {
		t.Fatalf("decodeAtlasMeta: %v", err)
	}
	if len(meta.Regions) != 1 {
		t.Fatalf("decoded %d regions, want 1", len(meta.Regions))
	}
	r := meta.Regions[0]
	if r.Name != "player" || r.X != 1 || r.Y != 2 || r.W != 3 || r.H != 4 {
		t.Fatalf("region = %+v, want {player 1 2 3 4}", r)
	}
}

func TestDecodeAtlasMetaEmptySidecar(t *testing.T) {
	meta, err := decodeAtlasMeta(strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("decodeAtlasMeta on an empty sidecar: %v", err)
	}
	if len(meta.Regions) != 0 {
		t.Fatalf("decoded %d regions from {}, want 0", len(meta.Regions))
	}
}

func TestDecodeAtlasMetaRejectsGarbage(t *testing.T) {
	if _, err := decodeAtlasMeta(strings.NewReader("not json")); err == nil {
		t.Fatal("decodeAtlasMeta on garbage input should error")
	}
}

func TestAtlasRegionRect(t *testing.T) {
	rect := AtlasRegion{X: 1, Y: 2, W: 3, H: 4}.Rect()
	if rect.Min.X != 1 || rect.Min.Y != 2 || rect.Max.X != 4 || rect.Max.Y != 6 {
		t.Fatalf("Rect() = %v, want (1,2)-(4,6)", rect)
	}
}

func TestNewAtlasValidates(t *testing.T) {
	good := map[string]AtlasRegion{"player": {X: 0, Y: 0, W: 4, H: 3}}
	cases := []struct {
		name    string
		id      AtlasID
		path    ID
		texW    int
		texH    int
		regions map[string]AtlasRegion
	}{
		{"empty id", "", "tex.png", 4, 3, good},
		{"empty texture path", "atlas", "", 4, 3, good},
		{"zero texture width", "atlas", "tex.png", 0, 3, good},
		{"zero texture height", "atlas", "tex.png", 4, 0, good},
		{"empty region name", "atlas", "tex.png", 4, 3, map[string]AtlasRegion{"": {X: 0, Y: 0, W: 1, H: 1}}},
		{"zero region width", "atlas", "tex.png", 4, 3, map[string]AtlasRegion{"player": {X: 0, Y: 0, W: 0, H: 3}}},
		{"region exceeds width", "atlas", "tex.png", 4, 3, map[string]AtlasRegion{"player": {X: 2, Y: 0, W: 3, H: 3}}},
		{"region exceeds height", "atlas", "tex.png", 4, 3, map[string]AtlasRegion{"player": {X: 0, Y: 1, W: 4, H: 3}}},
		{"negative region x", "atlas", "tex.png", 4, 3, map[string]AtlasRegion{"player": {X: -1, Y: 0, W: 2, H: 2}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewAtlas(c.id, c.path, c.texW, c.texH, c.regions); err == nil {
				t.Error("NewAtlas should reject invalid input")
			}
		})
	}
}

func TestNewAtlasCopiesRegionMap(t *testing.T) {
	regions := map[string]AtlasRegion{"player": {X: 0, Y: 0, W: 4, H: 3}}
	atlas, err := NewAtlas("atlas", "tex.png", 4, 3, regions)
	if err != nil {
		t.Fatalf("NewAtlas: %v", err)
	}
	regions["player"] = AtlasRegion{X: 99, Y: 99, W: 99, H: 99} // mutate the input
	got, err := atlas.Region("player")
	if err != nil {
		t.Fatalf("Region after input mutation: %v", err)
	}
	if got.X != 0 || got.W != 4 {
		t.Fatalf("atlas regions alias the caller's map: got %+v", got)
	}
}

func TestAtlasAccessorsAndLookup(t *testing.T) {
	atlas, err := NewAtlas("atlas", "tex.png", 8, 6, map[string]AtlasRegion{
		"player": {X: 0, Y: 0, W: 8, H: 6},
	})
	if err != nil {
		t.Fatalf("NewAtlas: %v", err)
	}
	if atlas.ID() != "atlas" || atlas.TexturePath() != "tex.png" || atlas.TextureW() != 8 || atlas.TextureH() != 6 {
		t.Fatalf("accessors = %q %q %dx%d", atlas.ID(), atlas.TexturePath(), atlas.TextureW(), atlas.TextureH())
	}
	if _, err := atlas.Region("player"); err != nil {
		t.Fatalf("Region hit: %v", err)
	}
	_, err = atlas.Region("enemy")
	if err == nil {
		t.Fatal("Region for a missing name should error")
	}
	if !strings.Contains(err.Error(), "enemy") || !strings.Contains(err.Error(), "atlas") {
		t.Errorf("error %v should name the region and the atlas", err)
	}
}

func TestAtlasStoreRegisterAndResolve(t *testing.T) {
	store := newStore()
	atlas, err := NewAtlas("atlas", "tex.png", 4, 3, map[string]AtlasRegion{
		"player": {X: 0, Y: 0, W: 4, H: 3},
	})
	if err != nil {
		t.Fatalf("NewAtlas: %v", err)
	}

	if err := store.Register(atlas); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := store.Register(atlas); err == nil {
		t.Error("duplicate registration should error")
	}
	if err := store.Register(nil); err == nil {
		t.Error("nil atlas registration should error")
	}

	resolved, err := store.Atlas("atlas")
	if err != nil {
		t.Fatalf("Atlas hit: %v", err)
	}
	if resolved != atlas {
		t.Fatal("Atlas returned a different instance")
	}
	if _, err := store.Atlas("other"); err == nil || !strings.Contains(err.Error(), "other") {
		t.Errorf("Atlas miss = %v, want an error naming the id", err)
	}

	region, err := store.Region("atlas", "player")
	if err != nil {
		t.Fatalf("Region hit: %v", err)
	}
	if region.W != 4 || region.H != 3 {
		t.Fatalf("region = %+v, want the registered rect", region)
	}
	_, err = store.Region("atlas", "enemy")
	if err == nil || !strings.Contains(err.Error(), "enemy") {
		t.Errorf("Region miss = %v, want an error naming the region", err)
	}
	_, err = store.Region("other", "player")
	if err == nil || !strings.Contains(err.Error(), "other") {
		t.Errorf("Region on unknown atlas = %v, want an error naming the atlas", err)
	}
}

func TestStoreIsHostedPerAsset(t *testing.T) {
	a := New(nil)
	store := a.Store()
	store2 := a.Store()
	if store != store2 {
		t.Fatal("Store() must return the same instance every call")
	}
	if b := New(nil); b.Store() == store {
		t.Fatal("two Assets must not share a store")
	}
}

func TestRegisterAtlasFromSidecar(t *testing.T) {
	a := New(newTestFS(map[string]string{
		"tex.png":    string(encodeTestPNG(t, 4, 3)),
		"atlas.json": `{"regions":[{"name":"player","x":0,"y":0,"w":4,"h":3}]}`,
	}))

	if err := a.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err != nil {
		t.Fatalf("RegisterAtlasFromSidecar: %v", err)
	}
	atlas, err := a.Store().Atlas("sprites")
	if err != nil {
		t.Fatalf("resolve registered atlas: %v", err)
	}
	if atlas.TexturePath() != "tex.png" || atlas.TextureW() != 4 || atlas.TextureH() != 3 {
		t.Fatalf("atlas = %q %dx%d, want tex.png 4x3", atlas.TexturePath(), atlas.TextureW(), atlas.TextureH())
	}
	region, err := a.Store().Region("sprites", "player")
	if err != nil {
		t.Fatalf("Region: %v", err)
	}
	if region != (AtlasRegion{X: 0, Y: 0, W: 4, H: 3}) {
		t.Fatalf("region = %+v", region)
	}

	// Failure leaves the store untouched: nothing registers under a
	// name whose sidecar is broken.
	bad := New(newTestFS(map[string]string{
		"tex.png":    string(encodeTestPNG(t, 4, 3)),
		"atlas.json": `{"regions":[{"name":"player","x":0,"y":0,"w":99,"h":3}]}`,
	}))
	if err := bad.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err == nil {
		t.Fatal("out-of-bounds sidecar region should error")
	}
	if _, err := bad.Store().Atlas("sprites"); err == nil {
		t.Fatal("a failed registration must leave the store untouched")
	}
}

func TestRegisterAtlasFromSidecarDuplicateRegion(t *testing.T) {
	a := New(newTestFS(map[string]string{
		"tex.png": string(encodeTestPNG(t, 4, 3)),
		"atlas.json": `{"regions":[` +
			`{"name":"player","x":0,"y":0,"w":2,"h":3},` +
			`{"name":"player","x":2,"y":0,"w":2,"h":3}]}`,
	}))
	if err := a.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err == nil {
		t.Fatal("duplicate region names in a sidecar should error")
	}
}

func TestRegisterAtlasFromSidecarMissingFile(t *testing.T) {
	a := New(newTestFS(map[string]string{
		"atlas.json": `{"regions":[]}`,
	}))
	err := a.RegisterAtlasFromSidecar("sprites", "missing.png", "atlas.json")
	if err == nil {
		t.Fatal("missing texture should error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %v should unwrap to fs.ErrNotExist", err)
	}
}

func TestRegisterGridAtlas(t *testing.T) {
	a := New(newTestFS(map[string]string{
		"tiles.png": string(encodeTestPNG(t, 4, 4)),
	}))

	if err := a.RegisterGridAtlas("tiles", "tiles.png", 2, 2, "tile"); err != nil {
		t.Fatalf("RegisterGridAtlas: %v", err)
	}
	want := map[string]AtlasRegion{
		"tile_0": {X: 0, Y: 0, W: 2, H: 2},
		"tile_1": {X: 2, Y: 0, W: 2, H: 2},
		"tile_2": {X: 0, Y: 2, W: 2, H: 2},
		"tile_3": {X: 2, Y: 2, W: 2, H: 2},
	}
	for name, region := range want {
		got, err := a.Store().Region("tiles", name)
		if err != nil {
			t.Fatalf("Region(%q): %v", name, err)
		}
		if got != region {
			t.Fatalf("Region(%q) = %+v, want %+v (row-major order)", name, got, region)
		}
	}

	// Uneven division, bad tile dims, and an empty prefix all error and
	// leave the store untouched.
	bad := New(newTestFS(map[string]string{"tiles.png": string(encodeTestPNG(t, 5, 4))}))
	if err := bad.RegisterGridAtlas("tiles", "tiles.png", 2, 2, "tile"); err == nil {
		t.Error("uneven division should error")
	}
	if err := bad.RegisterGridAtlas("tiles", "tiles.png", 0, 2, "tile"); err == nil {
		t.Error("zero tile width should error")
	}
	if err := bad.RegisterGridAtlas("tiles", "tiles.png", 2, 2, ""); err == nil {
		t.Error("empty prefix should error")
	}
	if _, err := bad.Store().Atlas("tiles"); err == nil {
		t.Error("failed registrations must leave the store untouched")
	}
}

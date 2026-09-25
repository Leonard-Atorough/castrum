package ebitrun

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

func newAssetTestFS(t *testing.T) fstest.MapFS {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return fstest.MapFS{
		"tex.png":    &fstest.MapFile{Data: buf.Bytes()},
		"atlas.json": &fstest.MapFile{Data: []byte(`{"regions":[{"name":"player","x":0,"y":0,"w":2,"h":2},{"name":"enemy","x":2,"y":2,"w":2,"h":2}]}`)},
	}
}

func TestNewWiresAssetPipeline(t *testing.T) {
	g, err := castrum.New()
	if err != nil {
		t.Fatalf("castrum.New: %v", err)
	}
	r, err := New(g, WithFilesystem(newAssetTestFS(t)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_ = r

	// The server and provider are locator resources, resolvable through
	// the world the way systems and DrawFuncs will resolve them.
	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("Resource[*asset.Server]: %v", err)
	}
	again, _ := g.World().Resource[*asset.Server]()
	if again != server {
		t.Fatal("Resource must return the same server instance")
	}

	provider, err := g.World().Resource[*TextureProvider]()
	if err != nil {
		t.Fatalf("Resource[*TextureProvider]: %v", err)
	}
	if provider.server != server {
		t.Fatal("provider must be wired to the same server")
	}
}

func TestNewWiresConfiguredFilesystem(t *testing.T) {
	g, _ := castrum.New()
	fs := newAssetTestFS(t)
	if _, err := New(g, WithFilesystem(fs)); err != nil {
		t.Fatalf("New: %v", err)
	}

	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("Resource[*asset.Server]: %v", err)
	}
	// The embedded-fs story: any fs.FS flows through, and paths resolve
	// against it.
	if _, err := server.Load[asset.TextureData]("tex.png"); err != nil {
		t.Fatalf("Load through the wired filesystem: %v", err)
	}
	if _, err := server.Load[asset.TextureData]("missing.png"); err == nil {
		t.Fatal("Load of a file outside the wired filesystem should error")
	}
}

func TestNewRejectsPreprovidedServer(t *testing.T) {
	g, _ := castrum.New()
	if err := g.World().Provide(func(*core.World) (*asset.Server, error) {
		return asset.New(nil), nil
	}); err != nil {
		t.Fatalf("pre-provide: %v", err)
	}
	if _, err := New(g); err == nil {
		t.Fatal("New should return the conflict when a server is already provided")
	}
}

func TestSubImageResolvesAndCaches(t *testing.T) {
	g, _ := castrum.New()
	if _, err := New(g, WithFilesystem(newAssetTestFS(t))); err != nil {
		t.Fatalf("New: %v", err)
	}

	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("Resource[*asset.Server]: %v", err)
	}
	if err := server.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err != nil {
		t.Fatalf("RegisterAtlasFromSidecar: %v", err)
	}
	provider, err := g.World().Resource[*TextureProvider]()
	if err != nil {
		t.Fatalf("Resource[*TextureProvider]: %v", err)
	}

	player, err := provider.SubImage("sprites", "player")
	if err != nil {
		t.Fatalf("SubImage: %v", err)
	}

	// Repeat resolution is a cache hit: the same pointer.
	again, err := provider.SubImage("sprites", "player")
	if err != nil {
		t.Fatalf("SubImage repeat: %v", err)
	}
	if again != player {
		t.Fatal("SubImage must return the same subimage on repeat calls")
	}

	// A different region of the same atlas shares the converted texture
	// and produces a distinct subimage.
	enemy, err := provider.SubImage("sprites", "enemy")
	if err != nil {
		t.Fatalf("SubImage second region: %v", err)
	}
	if enemy == player {
		t.Fatal("regions must resolve to distinct subimages")
	}
	if player.Bounds() == enemy.Bounds() {
		t.Fatal("distinct regions must have distinct bounds")
	}

	// Misses name their handles.
	_, err = provider.SubImage("sprites", "boss")
	if err == nil || !strings.Contains(err.Error(), "boss") {
		t.Errorf("unknown region error = %v, want it to name the region", err)
	}
	_, err = provider.SubImage("other", "player")
	if err == nil || !strings.Contains(err.Error(), "other") {
		t.Errorf("unknown atlas error = %v, want it to name the atlas", err)
	}
}

// Texture is the whole-texture half of the blit: convert once per path,
// repeat calls are the same image, misses error naming the path.
func TestTextureResolvesAndCaches(t *testing.T) {
	g, _ := castrum.New()
	if _, err := New(g, WithFilesystem(newAssetTestFS(t))); err != nil {
		t.Fatalf("New: %v", err)
	}
	provider, err := g.World().Resource[*TextureProvider]()
	if err != nil {
		t.Fatalf("Resource[*TextureProvider]: %v", err)
	}

	texture, err := provider.Texture("tex.png")
	if err != nil {
		t.Fatalf("Texture: %v", err)
	}
	if texture.Bounds() != image.Rect(0, 0, 4, 4) {
		t.Errorf("bounds = %v, want the full 4x4 texture", texture.Bounds())
	}

	again, err := provider.Texture("tex.png")
	if err != nil {
		t.Fatalf("Texture repeat: %v", err)
	}
	if again != texture {
		t.Fatal("Texture must return the same image on repeat calls")
	}

	_, err = provider.Texture("missing.png")
	if err == nil || !strings.Contains(err.Error(), "missing.png") {
		t.Errorf("unknown texture error = %v, want it to name the path", err)
	}
}

// SubImageRect is the region half of the blit: one subimage per path
// and rect, shared with the atlas-named path, with user-supplied rects
// validated against the texture.
func TestSubImageRectResolvesCachesAndValidates(t *testing.T) {
	g, _ := castrum.New()
	if _, err := New(g, WithFilesystem(newAssetTestFS(t))); err != nil {
		t.Fatalf("New: %v", err)
	}
	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("Resource[*asset.Server]: %v", err)
	}
	if err := server.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err != nil {
		t.Fatalf("RegisterAtlasFromSidecar: %v", err)
	}
	provider, err := g.World().Resource[*TextureProvider]()
	if err != nil {
		t.Fatalf("Resource[*TextureProvider]: %v", err)
	}

	player := image.Rect(0, 0, 2, 2)
	sub, err := provider.SubImageRect("tex.png", player)
	if err != nil {
		t.Fatalf("SubImageRect: %v", err)
	}
	if sub.Bounds() != player {
		t.Errorf("bounds = %v, want %v", sub.Bounds(), player)
	}

	// Repeat resolution is a cache hit: the same pointer.
	again, err := provider.SubImageRect("tex.png", player)
	if err != nil {
		t.Fatalf("SubImageRect repeat: %v", err)
	}
	if again != sub {
		t.Fatal("SubImageRect must return the same subimage on repeat calls")
	}

	// The atlas-named path resolves regions through the same cache.
	named, err := provider.SubImage("sprites", "player")
	if err != nil {
		t.Fatalf("SubImage: %v", err)
	}
	if named != sub {
		t.Fatal("SubImage and SubImageRect must share the cached subimage")
	}

	enemy := image.Rect(2, 2, 4, 4)
	other, err := provider.SubImageRect("tex.png", enemy)
	if err != nil {
		t.Fatalf("SubImageRect second rect: %v", err)
	}
	if other == sub || other.Bounds() != enemy {
		t.Fatal("distinct rects must resolve to distinct subimages with their own bounds")
	}

	// User-supplied rects are validated: empty and out-of-bounds rect
	// error instead of reaching the backend.
	_, err = provider.SubImageRect("tex.png", image.Rect(1, 1, 1, 1))
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty rect error = %v, want it to say empty", err)
	}
	_, err = provider.SubImageRect("tex.png", image.Rect(3, 3, 6, 6))
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("out-of-bounds rect error = %v, want it to say outside", err)
	}
	_, err = provider.SubImageRect("missing.png", player)
	if err == nil || !strings.Contains(err.Error(), "missing.png") {
		t.Errorf("unknown texture error = %v, want it to name the path", err)
	}
}

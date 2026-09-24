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

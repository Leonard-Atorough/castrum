package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

func TestAuthoredSheetMatchesShippedDocument(t *testing.T) {
	root := filepath.Join(t.TempDir(), "authored")
	name := filepath.Join(root, "gopher.data")
	if err := authorSheet(name); err != nil {
		t.Fatal(err)
	}
	directory, err := asset.NewDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	}()
	loader, err := asset.NewLoader(directory, asset.LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	authored, err := asset.FileSheet("gopher.data").Resolve(loader)
	if err != nil {
		t.Fatal(err)
	}
	content, err := asset.NewLoader(files, asset.LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	shipped, err := asset.FileSheet(sheetPath).Resolve(content)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(authored, shipped) {
		t.Fatalf("authored sheet=%+v, want shipped %+v", authored, shipped)
	}
	texture, err := content.Load[asset.TextureData](string(authored.Texture))
	if err != nil {
		t.Fatal(err)
	}
	if err := authored.ValidateBounds(texture.Image.Bounds()); err != nil {
		t.Fatal(err)
	}
	if _, err := authored.Region(wideRegion); err != nil {
		t.Fatalf("authored sheet lost the irregular region: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "gopher.data" {
		t.Fatalf("authoring should save only metadata: %v", entries)
	}
}

func TestAuthorSheetRejectsUnusableDestination(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, []byte("keep this file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := authorSheet(filepath.Join(parent, "gopher.json")); err == nil {
		t.Fatal("authoring reported success for a file used as the parent directory")
	}
	data, err := os.ReadFile(parent)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep this file" {
		t.Fatalf("authoring replaced the existing file: %q", data)
	}
}

func TestNewGameValidatesSheetAndSpawnsRegions(t *testing.T) {
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	const want = gridFrames + 1 // grid cells plus the wide region
	got := map[string]bool{}
	for e := range core.NewQuery(g.World()).With(sprite.Sprite{}, core.Transform{}).Execute() {
		s, _ := e.Component[sprite.Sprite]()
		if s.Asset != sheetPath {
			t.Errorf("entity %d asset = %q, want %q", e.ID(), s.Asset, sheetPath)
		}
		if s.Region == "" {
			t.Errorf("entity %d has no region; sheet sprites must name one", e.ID())
		}
		got[s.Region] = true
	}
	if len(got) != want {
		t.Fatalf("spawned regions = %v, want %d distinct", got, want)
	}
	for _, name := range append(slices.Clone(gridRegions), wideRegion) {
		if !got[name] {
			t.Errorf("region %q was not spawned", name)
		}
	}
	camera, ok := g.MainCamera().Component[core.Transform](g.World())
	if !ok || camera.Position != (geom.Vector2{X: screenW / 2, Y: screenH / 2}) {
		t.Fatalf("camera position = %v (ok=%t), want the screen center", camera.Position, ok)
	}
}

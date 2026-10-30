package builtin

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum/asset"
)

func TestRegisterLoadsBuiltinFontAndPreservesBaseAssets(t *testing.T) {
	base := fstest.MapFS{
		"sprites/player.txt": &fstest.MapFile{Data: []byte("player")},
	}
	filesystem := WithFilesystem(base)
	loader := asset.NewLoader(filesystem)

	if err := Register(loader); err != nil {
		t.Fatalf("register built-in font: %v", err)
	}

	font, err := loader.Load[asset.FontData](Font)
	if err != nil {
		t.Fatalf("load built-in font: %v", err)
	}
	if len(font.Raw) == 0 {
		t.Fatal("built-in font has no raw data")
	}

	data, err := fs.ReadFile(filesystem, "sprites/player.txt")
	if err != nil {
		t.Fatalf("read base filesystem asset: %v", err)
	}
	if string(data) != "player" {
		t.Fatalf("base filesystem asset = %q, want %q", data, "player")
	}
}

func TestRegisterRejectsNilLoader(t *testing.T) {
	if err := Register(nil); err == nil {
		t.Fatal("Register(nil) succeeded")
	}
}

func TestRegisterReturnsFontLoadError(t *testing.T) {
	if err := Register(asset.NewLoader(fstest.MapFS{})); err == nil {
		t.Fatal("Register succeeded without the built-in filesystem")
	}
}

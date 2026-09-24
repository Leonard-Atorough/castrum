package asset

import (
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

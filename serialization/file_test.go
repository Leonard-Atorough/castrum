package serialization

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/sprite"
)

func newFileTestRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}
	return reg
}

func TestOpenReturnsBuiltNodeWithChildren(t *testing.T) {
	reg := newFileTestRegistry(t)
	f := File{
		Key:        "root",
		Persistent: true,
		Components: []FileComponent{{Type: "game.Trader", Value: []byte(`{"name":"a"}`)}},
		Children:   []File{{Key: "kid"}},
	}

	n, err := f.Open(reg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if n.Key != "root" || !n.Persistent {
		t.Fatalf("node = key %q persistent %v", n.Key, n.Persistent)
	}
	if len(n.Components) != 1 || n.Components[0] != (trader{Name: "a"}) {
		t.Fatalf("components = %+v", n.Components)
	}
	if len(n.Children) != 1 || n.Children[0].Key != "kid" {
		t.Fatalf("children = %+v", n.Children)
	}
}

func TestOpenRejectsDuplicateComponentType(t *testing.T) {
	reg := newFileTestRegistry(t)
	f := File{
		Key: "root",
		Components: []FileComponent{
			{Type: "game.Trader", Value: []byte(`{"name":"a"}`)},
			{Type: "game.Trader", Value: []byte(`{"name":"b"}`)},
		},
	}

	_, err := f.Open(reg)
	if err == nil || !strings.Contains(err.Error(), "duplicate component type") {
		t.Fatalf("Open error = %v, want duplicate component type", err)
	}
}

func TestOpenRejectsUnknownComponentType(t *testing.T) {
	reg := newFileTestRegistry(t)
	f := File{
		Key:        "root",
		Components: []FileComponent{{Type: "game.Missing", Value: []byte(`{}`)}},
	}

	_, err := f.Open(reg)
	if err == nil || !strings.Contains(err.Error(), "unknown component type") {
		t.Fatalf("Open error = %v, want unknown component type", err)
	}
}

func shipNode() scene.Node {
	return scene.Node{
		Key:        "ship",
		Components: []any{trader{Name: "a", Patience: 1}, station{Credits: 5}},
	}
}

func TestSpawnRejectsNodeWithChildren(t *testing.T) {
	n := scene.Node{Key: "root", Children: []scene.Node{{Key: "kid"}}}

	_, err := Spawn(core.NewWorld(), n)
	if err == nil || !strings.Contains(err.Error(), "has children") {
		t.Fatalf("Spawn error = %v, want children rejection", err)
	}
}

func TestSpawnComponents(t *testing.T) {
	tests := []struct {
		name        string
		overrides   []any
		wantTrader  trader
		wantCredits int64
	}{
		{
			name:        "node components only",
			wantTrader:  trader{Name: "a", Patience: 1},
			wantCredits: 5,
		},
		{
			name:        "override replaces same type",
			overrides:   []any{trader{Name: "b", Patience: 9}},
			wantTrader:  trader{Name: "b", Patience: 9},
			wantCredits: 5,
		},
		{
			name:        "override adds new type",
			overrides:   []any{station{Credits: 7}},
			wantTrader:  trader{Name: "a", Patience: 1},
			wantCredits: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world := core.NewWorld()
			e, err := Spawn(world, shipNode(), tt.overrides...)
			if err != nil {
				t.Fatalf("Spawn: %v", err)
			}
			if got, ok := e.Component[trader](world); !ok || got != tt.wantTrader {
				t.Fatalf("trader = %+v, %v; want %+v", got, ok, tt.wantTrader)
			}
			if got, ok := e.Component[station](world); !ok || got.Credits != tt.wantCredits {
				t.Fatalf("station = %+v, %v; want credits %d", got, ok, tt.wantCredits)
			}
		})
	}
}

func TestSpawnPropagatesOverrideValidationError(t *testing.T) {
	_, err := Spawn(core.NewWorld(), shipNode(), trader{})
	if !errors.Is(err, errTraderName) {
		t.Fatalf("Spawn error = %v, want errTraderName", err)
	}
}

func TestOpenRejectsUnauthoredComponents(t *testing.T) {
	// Authored content must contribute something: engine visual
	// components implement core.Authored, and File.Open enforces it
	// with the node path named.
	reg := newFileTestRegistry(t)
	if err := reg.Component[sprite.Sprite]("sprite.Sprite"); err != nil {
		t.Fatalf("register sprite: %v", err)
	}
	if err := reg.Component[sprite.Text]("sprite.Text"); err != nil {
		t.Fatalf("register text: %v", err)
	}
	if err := reg.Component[sprite.Primitive]("sprite.Primitive"); err != nil {
		t.Fatalf("register primitive: %v", err)
	}

	for name, tc := range map[string]struct {
		key string
		raw string
	}{
		"sprite without asset":    {"sprite.Sprite", `{"style":{}}`},
		"text without font":       {"sprite.Text", `{"textSize":12}`},
		"primitive without shape": {"sprite.Primitive", `{"rect":{"x":1,"y":1}}`},
	} {
		f := File{Key: "root", Components: []FileComponent{{Type: tc.key, Value: []byte(tc.raw)}}}
		_, err := f.Open(reg)
		if err == nil || !strings.Contains(err.Error(), tc.key) || !strings.Contains(err.Error(), "root") {
			t.Errorf("%s: error = %v, want it to name key %q and node root", name, err, tc.key)
		}
	}
}

func TestOpenCompilesFileAssets(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}
	if err := reg.Asset[trader]("game.TraderAsset"); err != nil {
		t.Fatalf("Asset: %v", err)
	}

	f := File{
		Key: "root",
		Assets: []FileAsset{
			{Type: "game.TraderAsset", ID: "sprites.json"},
		},
		Children: []File{
			{Key: "kid", Assets: []FileAsset{{Type: "game.TraderAsset", ID: "font.ttf"}}},
		},
	}
	n, err := f.Open(reg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(n.Assets) != 1 {
		t.Fatalf("node assets = %d, want 1", len(n.Assets))
	}
	if len(n.Children) != 1 || len(n.Children[0].Assets) != 1 {
		t.Fatalf("child assets = %+v, want one child with one asset", n.Children)
	}
}

func TestOpenRejectsUnknownAssetType(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}

	f := File{
		Key:    "root",
		Assets: []FileAsset{{Type: "game.Mystery", ID: "x.png"}},
	}
	_, err := f.Open(reg)
	if err == nil || !strings.Contains(err.Error(), "game.Mystery") || !strings.Contains(err.Error(), "root") {
		t.Fatalf("Open error = %v, want it to name the asset key and node path", err)
	}
}

// TestFileShape pins the File format's JSON shape exactly, both
// directions.
//
// The authored half is what a game author types: field names, entry
// shapes, and the recursive node nesting. The encoded half is what a
// saver produces (today: round-trip only; a save writer arrives with
// runtime state serialization). Renaming any pinned field breaks
// authored files, so the pin fails loudly instead of silently.
//
// Deliberately not pinned: absence shapes (lists as null versus
// omitted) are saver decisions that do not exist yet, and
// unknown-field leniency is a policy the format has not declared.
func TestFileShape(t *testing.T) {
	const authored = `{
		"key": "root",
		"persistent": true,
		"components": [
			{ "type": "game.Trader", "value": { "name": "orsa" } },
			{ "type": "game.Station", "value": { "credits": 7 } }
		],
		"assets": [
			{ "type": "game.TraderAsset", "id": "sprites.json" }
		],
		"children": [
			{
				"key": "kid",
				"components": [
					{ "type": "game.Trader", "value": { "name": "pup" } }
				]
			}
		]
	}`

	t.Run("authored", func(t *testing.T) {
		var f File
		if err := json.Unmarshal([]byte(authored), &f); err != nil {
			t.Fatalf("unmarshal authored file: %v", err)
		}
		if f.Key != "root" || !f.Persistent {
			t.Fatalf("node header = %q persistent %v", f.Key, f.Persistent)
		}
		// Component values travel verbatim, authoring whitespace and
		// all; the pin compares the compacted JSON so it stays
		// readable.
		var compact bytes.Buffer
		if err := json.Compact(&compact, f.Components[0].Value); err != nil {
			t.Fatalf("compact component value: %v", err)
		}
		if len(f.Components) != 2 ||
			f.Components[0].Type != "game.Trader" || compact.String() != `{"name":"orsa"}` ||
			f.Components[1].Type != "game.Station" {
			t.Fatalf("component entries = %+v", f.Components)
		}
		// Verbatim passthrough is the contract: the authored spacing
		// above survives the decode untouched.
		if string(f.Components[0].Value) != `{ "name": "orsa" }` {
			t.Fatalf("value = %q, want the authored bytes verbatim", f.Components[0].Value)
		}
		if len(f.Assets) != 1 || f.Assets[0].Type != "game.TraderAsset" || f.Assets[0].ID != "sprites.json" {
			t.Fatalf("asset entries = %+v", f.Assets)
		}
		if len(f.Children) != 1 || f.Children[0].Key != "kid" || len(f.Children[0].Components) != 1 {
			t.Fatalf("children = %+v", f.Children)
		}

		// The file compiles through the registry like any authored file.
		reg := newFileTestRegistry(t)
		if err := reg.Component[station]("game.Station"); err != nil {
			t.Fatalf("Component: %v", err)
		}
		if err := reg.Asset[trader]("game.TraderAsset"); err != nil {
			t.Fatalf("Asset: %v", err)
		}
		if _, err := f.Open(reg); err != nil {
			t.Fatalf("Open authored file: %v", err)
		}
	})

	t.Run("encoded", func(t *testing.T) {
		f := File{
			Key:        "root",
			Persistent: true,
			Components: []FileComponent{{Type: "game.Trader", Value: []byte(`{"name":"orsa"}`)}},
			Assets:     []FileAsset{{Type: "game.TraderAsset", ID: "sprites.json"}},
			Children:   []File{{Key: "kid"}},
		}
		data, err := json.Marshal(f)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		// Component values pass through raw - verbatim JSON in,
		// verbatim JSON out - and children recurse as the same node
		// shape.
		const want = `{"key":"root","persistent":true,` +
			`"components":[{"type":"game.Trader","value":{"name":"orsa"}}],` +
			`"assets":[{"type":"game.TraderAsset","id":"sprites.json"}],` +
			`"children":[{"key":"kid","persistent":false,` +
			`"components":null,"assets":null,"children":null}]}`
		if strings.TrimSpace(string(data)) != want {
			t.Fatalf("encoded shape = %s, want %s", data, want)
		}
	})
}

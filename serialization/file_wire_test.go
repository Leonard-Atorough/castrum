package serialization

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestFileWireShape pins the File format's wire exactly, both directions.
//
// The authored half is what a game author types: field names, entry
// shapes, and the recursive node nesting. The emitted half is what a
// saver produces (today: round-trip only; a save writer arrives with
// runtime state serialization). Renaming any pinned field breaks
// authored files, so the pin fails loudly instead of silently.
//
// Deliberately not pinned: absence shapes (lists as null versus
// omitted) are saver decisions that do not exist yet, and unknown-field
// leniency is a policy the format has not declared.
func TestFileWireShape(t *testing.T) {
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
		// Values travel verbatim, authoring whitespace and all; the pin
		// compares the compacted JSON so it stays readable.
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

		// The wire compiles through the registry like any authored file.
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

	t.Run("emitted", func(t *testing.T) {
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
		// Component values travel verbatim - raw JSON in, raw JSON out -
		// and children recurse as the same node shape.
		const want = `{"key":"root","persistent":true,` +
			`"components":[{"type":"game.Trader","value":{"name":"orsa"}}],` +
			`"assets":[{"type":"game.TraderAsset","id":"sprites.json"}],` +
			`"children":[{"key":"kid","persistent":false,` +
			`"components":null,"assets":null,"children":null}]}`
		if strings.TrimSpace(string(data)) != want {
			t.Fatalf("wire = %s, want %s", data, want)
		}
	})
}

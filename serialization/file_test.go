package serialization

import (
	"errors"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/scene"
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

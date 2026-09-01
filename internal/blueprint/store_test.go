package blueprint

import (
	"os"
	"path/filepath"
	"testing"
)

const validBlueprintYAML = `
name: Goblin
version: "1.0"
tags:
  - enemy
  - hostile
components:
  - type: testComponent
    properties:
      Value: 5
`

func TestStore_LoadFromString(t *testing.T) {
	t.Run("parses a valid blueprint and registers it in the store", func(t *testing.T) {
		s := NewStore()

		bp, err := s.LoadFromString(validBlueprintYAML)
		if err != nil {
			t.Fatalf("LoadFromString failed: %v", err)
		}
		if bp.Name != "Goblin" {
			t.Fatalf("Name = %q, want %q", bp.Name, "Goblin")
		}
		if len(bp.Components) != 1 || bp.Components[0].Type != "testComponent" {
			t.Fatalf("unexpected components: %#v", bp.Components)
		}

		got, exists := s.GetBlueprint("Goblin")
		if !exists || got != bp {
			t.Fatal("expected Load to register the blueprint under its name")
		}
	})

	t.Run("invalid YAML returns an error", func(t *testing.T) {
		s := NewStore()
		if _, err := s.LoadFromString("not: [valid"); err == nil {
			t.Fatal("expected an error for malformed YAML")
		}
	})
}

func TestStore_LoadFromBytes(t *testing.T) {
	s := NewStore()
	bp, err := s.LoadFromBytes([]byte(validBlueprintYAML))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}
	if bp.Name != "Goblin" {
		t.Fatalf("Name = %q, want %q", bp.Name, "Goblin")
	}
}

func TestStore_LoadFromPath(t *testing.T) {
	t.Run("loads a blueprint from a file on disk", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "goblin.yaml")
		if err := os.WriteFile(path, []byte(validBlueprintYAML), 0o644); err != nil {
			t.Fatalf("failed to write test fixture: %v", err)
		}

		s := NewStore()
		bp, err := s.LoadFromPath(path)
		if err != nil {
			t.Fatalf("LoadFromPath failed: %v", err)
		}
		if bp.Name != "Goblin" {
			t.Fatalf("Name = %q, want %q", bp.Name, "Goblin")
		}
	})

	t.Run("a missing file returns an error", func(t *testing.T) {
		s := NewStore()
		if _, err := s.LoadFromPath(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
			t.Fatal("expected an error for a missing file")
		}
	})
}

func TestStore_AddAndGetBlueprint(t *testing.T) {
	s := NewStore()
	bp := &Blueprint{Name: "Orc"}

	s.AddBlueprint(bp.Name, bp)

	got, exists := s.GetBlueprint("Orc")
	if !exists || got != bp {
		t.Fatal("expected to retrieve the added blueprint")
	}

	if _, exists := s.GetBlueprint("nonexistent"); exists {
		t.Fatal("expected no blueprint for an unregistered name")
	}
}

package castrum

import (
	"errors"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/core"
)

// WithResource must register a validated resource at construction, panic in
// New on an invalid one, and AddResource must reject invalid values without
// touching the world.
func TestWithResourceRegistersAndValidates(t *testing.T) {
	sets, err := animation.NewSets(map[string]animation.Set{
		"hero": {Name: "hero", States: map[string]animation.State{
			"idle": {Name: "idle", Loop: animation.Loop, Tracks: []animation.Track{
				{Name: "body", Source: "atlas.json", Frames: []string{"a"}, FPS: 10},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	g := New(WithResource(sets))
	got, err := g.World().Resource[animation.Sets]()
	if err != nil || got.Names()[0] != "hero" {
		t.Fatalf("Resource[Sets] = %v, %v; want the registered hero set", got, err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("New(WithResource(invalid)) did not panic")
		}
	}()
	New(WithResource(animation.Sets{})) // zero Sets fails Validate
}

func TestAddResourceValidates(t *testing.T) {
	g := New()
	if err := g.InsertResource(animation.Sets{}); err == nil || !strings.Contains(err.Error(), "at least one set") {
		t.Fatalf("AddResource(invalid) = %v, want validation error", err)
	}
	if _, err := g.World().Resource[animation.Sets](); !errors.Is(err, core.ErrResourceNotRegistered) {
		t.Fatalf("invalid AddResource registered a resource: %v", err)
	}
}

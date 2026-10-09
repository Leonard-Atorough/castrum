package collision

import (
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestColliderValidate(t *testing.T) {
	valid := []Collider{
		{
			Shape:   Box{Min: geom.Vector2{X: -1, Y: -2}, Max: geom.Vector2{X: 3, Y: 4}},
			Offset:  geom.Vector2{X: 2, Y: 5},
			Layers:  1<<0 | 1<<31,
			Mask:    1 << 31,
			Trigger: true,
			Active:  true,
		},
		// A collider that belongs to no layer is legal: it collides
		// with nothing, the mirror of an empty mask.
		{Shape: Circle{Center: geom.Vector2{X: 1, Y: 1}, Radius: 8}, Active: true},
	}
	for _, collider := range valid {
		if err := collider.Validate(); err != nil {
			t.Fatalf("valid Collider rejected: %v", err)
		}
	}

	nan := math.NaN()
	inf := math.Inf(1)
	for name, collider := range map[string]Collider{
		"nil shape":              {},
		"inverted rect":          {Shape: Box{Min: geom.Vector2{X: 4, Y: 4}, Max: geom.Vector2{X: 0, Y: 2}}},
		"zero-area rect":         {Shape: Box{}},
		"rect NaN bound":         {Shape: Box{Min: geom.Vector2{X: nan}, Max: geom.Vector2{X: 2, Y: 2}}},
		"rect infinite bound":    {Shape: Box{Min: geom.Vector2{X: -inf}, Max: geom.Vector2{X: 2, Y: 2}}},
		"circle zero radius":     {Shape: Circle{}},
		"circle negative radius": {Shape: Circle{Radius: -1}},
		"circle NaN radius":      {Shape: Circle{Radius: nan}},
		"circle NaN center":      {Shape: Circle{Center: geom.Vector2{X: nan}, Radius: 1}},
		"NaN offset":             {Shape: Circle{Radius: 1}, Offset: geom.Vector2{X: nan}},
	} {
		if err := collider.Validate(); err == nil {
			t.Errorf("Collider with %s should fail validation", name)
		}
	}
}

func TestNewCollider(t *testing.T) {
	collider, err := NewCollider(Circle{Center: geom.Vector2{X: 10, Y: 10}, Radius: 4})
	if err != nil {
		t.Fatalf("NewCollider rejected a valid circle: %v", err)
	}
	if !collider.Active {
		t.Error("NewCollider collider should be active")
	}
	if collider.Mask != math.MaxUint32 {
		t.Errorf("NewCollider mask = %#x, want every layer open", collider.Mask)
	}
	if collider.Layers != 1 || collider.Trigger || collider.Offset != (geom.Vector2{}) {
		t.Errorf("NewCollider defaults not as expected: %+v", collider)
	}

	if _, err := NewCollider(nil); err == nil {
		t.Error("NewCollider should reject a nil shape")
	}
	if _, err := NewCollider(Box{}); err == nil {
		t.Error("NewCollider should reject a degenerate rect")
	}
}

func TestColliderCanCollideWith(t *testing.T) {
	player := Collider{Layers: 1 << 0, Mask: 1<<0 | 1<<2}
	wall := Collider{Layers: 1 << 2, Mask: 1<<0 | 1<<2}
	oneSided := Collider{Layers: 1 << 2, Mask: 1 << 2}
	closed := Collider{Layers: 1 << 0, Mask: 0}
	open := Collider{Layers: 1 << 3, Mask: math.MaxUint32}
	// Cross-listening: neither shares a layer it both sits on and
	// listens to, yet each listens to a layer the other sits on.
	crossA := Collider{Layers: 1 << 0, Mask: 1 << 2}
	crossB := Collider{Layers: 1 << 2, Mask: 1 << 0}
	// Composition: a collider sitting on several layers matches a
	// mask that admits any one of them.
	hazardAndSolid := Collider{Layers: 1<<2 | 1<<3, Mask: 1 << 0}

	if !player.CanCollideWith(wall) || !wall.CanCollideWith(player) {
		t.Error("two colliders listing each other's layers should collide")
	}
	if player.CanCollideWith(oneSided) || oneSided.CanCollideWith(player) {
		t.Error("a one-sided mask should not collide")
	}
	if closed.CanCollideWith(open) || open.CanCollideWith(closed) {
		t.Error("an empty mask should not collide with anything")
	}
	if !crossA.CanCollideWith(crossB) || !crossB.CanCollideWith(crossA) {
		t.Error("cross-listening colliders should collide")
	}
	if !hazardAndSolid.CanCollideWith(Collider{Layers: 1 << 0, Mask: 1 << 3}) {
		t.Error("a multi-layer collider should match through any shared layer")
	}
	// Belonging to no layer collides with nothing, like listening to
	// none.
	layerless := Collider{Layers: 0, Mask: math.MaxUint32}
	if layerless.CanCollideWith(open) || open.CanCollideWith(layerless) {
		t.Error("a collider on no layer should not collide with anything")
	}
}

// The Validatable hook fires through the public surface: spawn rejects
// an invalid collider, and the NewCollider defaults survive a spawn.
func TestColliderValidationAtSpawn(t *testing.T) {
	w := core.NewWorld()

	if _, err := w.NewEntity(Collider{Shape: Circle{Radius: -8}}); err == nil ||
		!strings.Contains(err.Error(), "Collider") {
		t.Errorf("negative radius at spawn = %v, want an error naming Collider", err)
	}
	if _, err := w.NewEntity(Collider{}); err == nil {
		t.Error("nil shape at spawn should error")
	}

	collider, err := NewCollider(Circle{Radius: 8})
	if err != nil {
		t.Fatalf("NewCollider: %v", err)
	}
	if _, err := w.NewEntity(collider); err != nil {
		t.Errorf("default collider should spawn: %v", err)
	}
}

// The layer-number helpers build the bitmasks beginners read, and
// reject the indexes whose bits would silently shift out.
func TestLayersAndMaskHelpers(t *testing.T) {
	if got := Layers(); got != 0 {
		t.Errorf("Layers() = %#x, want the empty bitmask", got)
	}
	if got := Mask(); got != 0 {
		t.Errorf("Mask() = %#x, want the empty mask", got)
	}
	if got := Layers(0, 2, 2); got != 1<<0|1<<2 {
		t.Errorf("Layers(0, 2, 2) = %#x, want layers 0 and 2", got)
	}
	if got := Mask(0); got != 1<<0 {
		t.Errorf("Mask(0) = %#x, want layer 0 alone", got)
	}

	for name, layer := range map[string]int{"negative": -1, "over 31": 32} {
		for _, helper := range []struct {
			name string
			call func(int) uint32
		}{
			{"Layers", func(l int) uint32 { return Layers(l) }},
			{"Mask", func(l int) uint32 { return Mask(l) }},
		} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("%s with %s layer %d should panic", helper.name, name, layer)
					}
				}()
				_ = helper.call(layer)
			}()
		}
	}
}

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
			Shape:   RectShape{Min: geom.Vector2{X: -1, Y: -2}, Max: geom.Vector2{X: 3, Y: 4}},
			Offset:  geom.Vector2{X: 2, Y: 5},
			Layer:   31,
			Mask:    1 << 31,
			Trigger: true,
			Active:  true,
		},
		{Shape: CircleShape{Center: geom.Vector2{X: 1, Y: 1}, Radius: 8}, Active: true},
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
		"inverted rect":          {Shape: RectShape{Min: geom.Vector2{X: 4, Y: 4}, Max: geom.Vector2{X: 0, Y: 2}}},
		"zero-area rect":         {Shape: RectShape{}},
		"rect NaN bound":         {Shape: RectShape{Min: geom.Vector2{X: nan}, Max: geom.Vector2{X: 2, Y: 2}}},
		"rect infinite bound":    {Shape: RectShape{Min: geom.Vector2{X: -inf}, Max: geom.Vector2{X: 2, Y: 2}}},
		"circle zero radius":     {Shape: CircleShape{}},
		"circle negative radius": {Shape: CircleShape{Radius: -1}},
		"circle NaN radius":      {Shape: CircleShape{Radius: nan}},
		"circle NaN center":      {Shape: CircleShape{Center: geom.Vector2{X: nan}, Radius: 1}},
		"layer over 31":          {Shape: CircleShape{Radius: 1}, Layer: 32},
		"NaN offset":             {Shape: CircleShape{Radius: 1}, Offset: geom.Vector2{X: nan}},
	} {
		if err := collider.Validate(); err == nil {
			t.Errorf("Collider with %s should fail validation", name)
		}
	}
}

func TestNewCollider(t *testing.T) {
	collider, err := NewCollider(CircleShape{Center: geom.Vector2{X: 10, Y: 10}, Radius: 4})
	if err != nil {
		t.Fatalf("NewCollider rejected a valid circle: %v", err)
	}
	if !collider.Active {
		t.Error("NewCollider collider should be active")
	}
	if collider.Mask != math.MaxUint32 {
		t.Errorf("NewCollider mask = %#x, want every layer open", collider.Mask)
	}
	if collider.Layer != 0 || collider.Trigger || collider.Offset != (geom.Vector2{}) {
		t.Errorf("NewCollider defaults not zeroed: %+v", collider)
	}

	if _, err := NewCollider(nil); err == nil {
		t.Error("NewCollider should reject a nil shape")
	}
	if _, err := NewCollider(RectShape{}); err == nil {
		t.Error("NewCollider should reject a degenerate rect")
	}
}

func TestColliderCanCollideWith(t *testing.T) {
	player := Collider{Layer: 0, Mask: 1<<0 | 1<<2}
	wall := Collider{Layer: 2, Mask: 1<<0 | 1<<2}
	oneSided := Collider{Layer: 2, Mask: 1 << 2}
	closed := Collider{Layer: 0, Mask: 0}
	open := Collider{Layer: 3, Mask: math.MaxUint32}

	if !player.CanCollideWith(wall) || !wall.CanCollideWith(player) {
		t.Error("two colliders listing each other's layers should collide")
	}
	if player.CanCollideWith(oneSided) || oneSided.CanCollideWith(player) {
		t.Error("a one-sided mask should not collide")
	}
	if closed.CanCollideWith(open) || open.CanCollideWith(closed) {
		t.Error("an empty mask should not collide with anything")
	}
}

// The Validatable hook fires through the public surface: spawn rejects
// an invalid collider, and the NewCollider defaults survive a spawn.
func TestColliderValidationAtSpawn(t *testing.T) {
	w := core.NewWorld()

	if _, err := w.NewEntity(Collider{Shape: CircleShape{Radius: 8}, Layer: 32}); err == nil ||
		!strings.Contains(err.Error(), "Collider") {
		t.Errorf("layer 32 at spawn = %v, want an error naming Collider", err)
	}
	if _, err := w.NewEntity(Collider{}); err == nil {
		t.Error("nil shape at spawn should error")
	}

	collider, err := NewCollider(CircleShape{Radius: 8})
	if err != nil {
		t.Fatalf("NewCollider: %v", err)
	}
	if _, err := w.NewEntity(collider); err != nil {
		t.Errorf("default collider should spawn: %v", err)
	}
}

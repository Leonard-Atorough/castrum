package collision

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestTransformShape_RectRotationBuildsConservativeBounds(t *testing.T) {
	shape := RectShape{Min: geom.Vector2{X: -2, Y: -1}, Max: geom.Vector2{X: 2, Y: 1}}

	rotated90 := transformShape(shape, geom.Vector2{}, core.Transform{Rotation: math.Pi / 2})
	if got := rotated90.bounds.Width(); math.Abs(got-2) > 1e-9 {
		t.Errorf("90-degree bounds width = %v, want 2", got)
	}
	if got := rotated90.bounds.Height(); math.Abs(got-4) > 1e-9 {
		t.Errorf("90-degree bounds height = %v, want 4", got)
	}

	rotated45 := transformShape(shape, geom.Vector2{}, core.Transform{Rotation: math.Pi / 4})
	wantExtent := 3 * math.Sqrt(2)
	if got := rotated45.bounds.Width(); math.Abs(got-wantExtent) > 1e-9 {
		t.Errorf("45-degree bounds width = %v, want %v", got, wantExtent)
	}
	if got := rotated45.bounds.Height(); math.Abs(got-wantExtent) > 1e-9 {
		t.Errorf("45-degree bounds height = %v, want %v", got, wantExtent)
	}
}

func TestTransformShape_PreservesLocalOffsetThroughRotation(t *testing.T) {
	shape := CircleShape{Center: geom.Vector2{X: 2, Y: 0}, Radius: 1}
	transformed := transformShape(shape, geom.Vector2{}, core.Transform{
		Position: geom.Vector2{X: 10, Y: 5},
		Rotation: math.Pi / 2,
	})

	circle, ok := transformed.shape.(worldCircle)
	if !ok {
		t.Fatalf("transformed shape type = %T, want worldCircle", transformed.shape)
	}
	if math.Abs(circle.center.X-10) > 1e-9 || math.Abs(circle.center.Y-7) > 1e-9 {
		t.Errorf("transformed circle center = %v, want (10, 7)", circle.center)
	}
}

func TestTransformShape_AppliesColliderOffsetBeforeRotation(t *testing.T) {
	transformed := transformShape(CircleShape{Radius: 1}, geom.Vector2{X: 3, Y: 0}, core.Transform{
		Position: geom.Vector2{X: 1, Y: 1},
		Rotation: math.Pi / 2,
	})

	circle, ok := transformed.shape.(worldCircle)
	if !ok {
		t.Fatalf("transformed shape type = %T, want worldCircle", transformed.shape)
	}
	if math.Abs(circle.center.X-1) > 1e-9 || math.Abs(circle.center.Y-4) > 1e-9 {
		t.Errorf("offset circle center = %v, want (1, 4)", circle.center)
	}
}

func TestTransformShape_CircleIgnoresScale(t *testing.T) {
	transformed := transformShape(CircleShape{Radius: 2}, geom.Vector2{}, core.Transform{
		Scale: geom.Vector2{X: 2, Y: 3},
	})

	circle, ok := transformed.shape.(worldCircle)
	if !ok {
		t.Fatalf("transformed shape type = %T, want worldCircle", transformed.shape)
	}
	if circle.radius != 2 {
		t.Errorf("transformed circle radius = %v, want 2 (scale must not affect colliders)", circle.radius)
	}
	if got := transformed.bounds.Width(); math.Abs(got-4) > 1e-9 {
		t.Errorf("transformed bounds width = %v, want 4 (bounds follow the shape, not the scale)", got)
	}
}

func TestContactBetween_RotatedRectanglesUseOrientedGeometry(t *testing.T) {
	shape := RectShape{Min: geom.Vector2{X: -2, Y: -0.5}, Max: geom.Vector2{X: 2, Y: 0.5}}
	verticalA := transformShape(shape, geom.Vector2{}, core.Transform{Rotation: math.Pi / 2})
	verticalB := transformShape(shape, geom.Vector2{}, core.Transform{
		Position: geom.Vector2{Y: 3.5},
		Rotation: math.Pi / 2,
	})

	if !contactBetween(verticalA.shape, verticalB.shape).collided {
		t.Fatal("expected vertically aligned rotated rectangles to collide")
	}
}

func TestContactBetween_CircleCircleNormalPointsFirstToSecond(t *testing.T) {
	a := transformShape(CircleShape{Radius: 2}, geom.Vector2{}, core.Transform{})
	b := transformShape(CircleShape{Radius: 2}, geom.Vector2{}, core.Transform{Position: geom.Vector2{X: 3, Y: 0}})

	hit := contactBetween(a.shape, b.shape)
	if !hit.collided {
		t.Fatal("overlapping circles should collide")
	}
	if math.Abs(hit.penetration-1) > 1e-9 {
		t.Errorf("penetration = %v, want 1", hit.penetration)
	}
	if math.Abs(hit.normal.X-1) > 1e-9 || math.Abs(hit.normal.Y) > 1e-9 {
		t.Errorf("normal = %v, want (1, 0)", hit.normal)
	}

	// The same pair in the other order mirrors the normal.
	flipped := contactBetween(b.shape, a.shape)
	if math.Abs(flipped.normal.X+1) > 1e-9 || math.Abs(flipped.normal.Y) > 1e-9 {
		t.Errorf("flipped normal = %v, want (-1, 0)", flipped.normal)
	}
}

func TestContactBetween_CircleRectNormalPointsFirstToSecond(t *testing.T) {
	circle := transformShape(CircleShape{Center: geom.Vector2{X: 5, Y: 0}, Radius: 1}, geom.Vector2{}, core.Transform{})
	rect := transformShape(RectShape{Min: geom.Vector2{X: 2, Y: -1}, Max: geom.Vector2{X: 4, Y: 1}}, geom.Vector2{}, core.Transform{})

	// The circle touches the rect's edge: distance from (5,0) to the
	// rect equals the radius, and touching counts.
	hit := contactBetween(circle.shape, rect.shape)
	if !hit.collided {
		t.Fatal("circle touching the rect edge should collide")
	}
	if math.Abs(hit.penetration) > 1e-9 {
		t.Errorf("touching penetration = %v, want 0", hit.penetration)
	}
	if math.Abs(hit.normal.X+1) > 1e-9 || math.Abs(hit.normal.Y) > 1e-9 {
		t.Errorf("normal = %v, want (-1, 0) pointing from the circle toward the rect", hit.normal)
	}
	if math.Abs(hit.point.X-4) > 1e-9 || math.Abs(hit.point.Y) > 1e-9 {
		t.Errorf("point = %v, want (4, 0)", hit.point)
	}

	// The rect-first order mirrors the same pair's normal.
	flipped := contactBetween(rect.shape, circle.shape)
	if math.Abs(flipped.normal.X-1) > 1e-9 || math.Abs(flipped.normal.Y) > 1e-9 {
		t.Errorf("flipped normal = %v, want (1, 0)", flipped.normal)
	}
}

// A circle contained in a rect reports the full depth to separate -
// boundary distance plus the radius - with the normal pointing at
// the nearest face, and the exit through that face reads cleanly.
func TestContactBetween_ContainedCircleDepth(t *testing.T) {
	rect := transformShape(RectShape{Min: geom.Vector2{X: -2, Y: -2}, Max: geom.Vector2{X: 2, Y: 2}}, geom.Vector2{}, core.Transform{})

	// Center one unit right of the rect's center: the nearest face is
	// the right wall, one unit away.
	contained := transformShape(CircleShape{Center: geom.Vector2{X: 1, Y: 0}, Radius: 0.5}, geom.Vector2{}, core.Transform{})
	hit := contactBetween(contained.shape, rect.shape)
	if !hit.collided {
		t.Fatal("contained circle should collide")
	}
	if math.Abs(hit.penetration-1.5) > 1e-9 {
		t.Errorf("penetration = %v, want 1.5 (boundary distance 1 plus radius 0.5)", hit.penetration)
	}
	if hit.normal != (geom.Vector2{X: 1, Y: 0}) {
		t.Errorf("normal = %v, want (1, 0) toward the nearest face", hit.normal)
	}
	if hit.point != (geom.Vector2{X: 2, Y: 0}) {
		t.Errorf("point = %v, want (2, 0) on the nearest face", hit.point)
	}

	// A center exactly on the boundary is the depth-zero case of the
	// same rule: the penetration is the radius alone.
	onEdge := transformShape(CircleShape{Center: geom.Vector2{X: 2, Y: 0}, Radius: 0.5}, geom.Vector2{}, core.Transform{})
	hit = contactBetween(onEdge.shape, rect.shape)
	if !hit.collided {
		t.Fatal("circle centered on the rect's edge should collide")
	}
	if math.Abs(hit.penetration-0.5) > 1e-9 {
		t.Errorf("on-boundary penetration = %v, want 0.5 (the radius)", hit.penetration)
	}
}

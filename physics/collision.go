package physics

import (
	"fmt"
	"math"
	"strings"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
)

// CollisionState stores the last exact result for a currently colliding pair.
type CollisionState struct {
	CollisionResult
	WasColliding bool
}

// CollisionErrors accumulates errors encountered while testing a batch of
// collision pairs, allowing the caller to report all failures together.
type CollisionErrors struct {
	Errors []error
}

// Add appends a non-nil error to the collection.
func (ce *CollisionErrors) Add(err error) {
	if err != nil {
		ce.Errors = append(ce.Errors, err)
	}
}

// Error returns a formatted summary of all accumulated errors.
func (ce *CollisionErrors) Error() string {
	if len(ce.Errors) == 0 {
		return ""
	}
	if len(ce.Errors) == 1 {
		return ce.Errors[0].Error()
	}
	var msg strings.Builder
	fmt.Fprintf(&msg, "collision: %d errors: ", len(ce.Errors))
	for i, err := range ce.Errors {
		if i > 0 {
			msg.WriteString("; ")
		}
		msg.WriteString(err.Error())
	}
	return msg.String()
}

// Unwrap exposes the accumulated errors to errors.Is and errors.As.
func (ce *CollisionErrors) Unwrap() []error {
	return ce.Errors
}

// IsEmpty reports whether no errors have been accumulated.
func (ce *CollisionErrors) IsEmpty() bool {
	return len(ce.Errors) == 0
}

// transformedShape keeps the exact world-space shape alongside its conservative
// AABB. The exact shape is for narrow-phase tests; the AABB is for indexing.
type transformedShape struct {
	shape  any
	bounds geom.Rect
}

func intersectsAny(shapeA, shapeB any) CollisionResult {
	switch a := shapeA.(type) {
	case orientedRect:
		switch b := shapeB.(type) {
		case orientedRect:
			return orientedRectContact(a, b)
		case geom.Circle:
			return circleOrientedRectContact(b, a)
		}
	case geom.Circle:
		switch b := shapeB.(type) {
		case orientedRect:
			return circleOrientedRectContact(a, b)
		case geom.Circle:
			return circleCircleContact(a, b)
		}
	}
	return CollisionResult{}
}

func circleCircleContact(a, b geom.Circle) CollisionResult {
	dx := b.Center.X - a.Center.X
	dy := b.Center.Y - a.Center.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	minDist := a.Radius + b.Radius

	if dist > minDist {
		return CollisionResult{Collided: false}
	}

	if dist == 0 {
		// Coincident centers have no unique normal. Use a stable fallback.
		return CollisionResult{
			Collided:    true,
			Penetration: minDist,
			Normal:      geom.Vector2{X: 1, Y: 0},
			Point:       a.Center,
		}
	}

	nx := dx / dist
	ny := dy / dist
	return CollisionResult{
		Collided:    true,
		Penetration: minDist - dist,
		Normal:      geom.Vector2{X: nx, Y: ny},
		Point:       geom.Vector2{X: a.Center.X + nx*a.Radius, Y: a.Center.Y + ny*a.Radius},
	}
}

type orientedRect struct {
	center      geom.Vector2    // World-space center.
	halfExtents geom.Vector2    // Half-width and half-height before rotation.
	axes        [2]geom.Vector2 // Orthonormal world-space local axes.
	bounds      geom.Rect       // Conservative world-space AABB.
}

func circleOrientedRectContact(circle geom.Circle, rect orientedRect) CollisionResult {
	local := circle.Center.Sub(rect.center)
	projectionX := local.Dot(rect.axes[0])
	projectionY := local.Dot(rect.axes[1])
	closestLocal := geom.Vector2{
		X: math.Max(-rect.halfExtents.X, math.Min(projectionX, rect.halfExtents.X)),
		Y: math.Max(-rect.halfExtents.Y, math.Min(projectionY, rect.halfExtents.Y)),
	}
	closest := rect.center.Add(rect.axes[0].Mul(closestLocal.X)).Add(rect.axes[1].Mul(closestLocal.Y))

	dx := circle.Center.X - closest.X
	dy := circle.Center.Y - closest.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > circle.Radius {
		return CollisionResult{Collided: false}
	}

	if dist == 0 {
		// The closest point is the circle center when it is inside the rectangle,
		// so there is no unique contact normal. Use a stable fallback direction.
		return CollisionResult{
			Collided:    true,
			Penetration: circle.Radius,
			Normal:      geom.Vector2{X: 1, Y: 0},
			Point:       circle.Center,
		}
	}

	nx := dx / dist
	ny := dy / dist
	return CollisionResult{
		Collided:    true,
		Penetration: circle.Radius - dist,
		Normal:      geom.Vector2{X: nx, Y: ny},
		Point:       closest,
	}
}

func orientedRectContact(a, b orientedRect) CollisionResult {
	axes := [4]geom.Vector2{a.axes[0], a.axes[1], b.axes[0], b.axes[1]}
	minimumOverlap := math.Inf(1)
	minimumAxis := geom.Vector2{}
	centerDelta := b.center.Sub(a.center)

	for _, axis := range axes {
		centerDistance := math.Abs(centerDelta.Dot(axis))
		radiusA := a.halfExtents.X*math.Abs(a.axes[0].Dot(axis)) +
			a.halfExtents.Y*math.Abs(a.axes[1].Dot(axis))
		radiusB := b.halfExtents.X*math.Abs(b.axes[0].Dot(axis)) +
			b.halfExtents.Y*math.Abs(b.axes[1].Dot(axis))
		overlap := radiusA + radiusB - centerDistance
		if overlap < 0 {
			return CollisionResult{Collided: false}
		}
		if overlap < minimumOverlap {
			minimumOverlap = overlap
			minimumAxis = axis
		}
	}

	if centerDelta.Dot(minimumAxis) < 0 {
		minimumAxis = minimumAxis.Mul(-1)
	}
	return CollisionResult{
		Collided:    true,
		Penetration: minimumOverlap,
		Normal:      minimumAxis,
		Point:       a.center.Add(minimumAxis.Mul(a.halfExtents.X)),
	}
}

// transformedCollider applies scale, rotation, and translation to a local
// collider. Rectangles retain an oriented representation for exact SAT tests;
// both supported shapes also receive a conservative world-space AABB.
func transformedCollider(shape any, transform components.Transform) (transformedShape, error) {
	if shape == nil {
		return transformedShape{}, fmt.Errorf("collider shape is nil")
	}

	switch local := shape.(type) {
	case geom.Rect:
		corners := [4]geom.Vector2{
			{X: local.Min.X, Y: local.Min.Y},
			{X: local.Max.X, Y: local.Min.Y},
			{X: local.Max.X, Y: local.Max.Y},
			{X: local.Min.X, Y: local.Max.Y},
		}
		worldCorners := transformPoints(corners, transform)
		center := transformPoint(local.Min.Add(local.Max).Mul(0.5), transform)
		scale := effectiveScale(transform.Scale)
		halfExtents := geom.Vector2{
			X: math.Abs(local.Width()*scale.X) / 2,
			Y: math.Abs(local.Height()*scale.Y) / 2,
		}
		oriented := orientedRect{
			center:      center,
			halfExtents: halfExtents,
			axes: [2]geom.Vector2{
				{X: math.Cos(transform.Rotation), Y: math.Sin(transform.Rotation)},
				{X: -math.Sin(transform.Rotation), Y: math.Cos(transform.Rotation)},
			},
			bounds: boundsOf(worldCorners[:]),
		}
		return transformedShape{shape: oriented, bounds: oriented.bounds}, nil
	case geom.Circle:
		center := transformPoint(local.Center, transform)
		scale := effectiveScale(transform.Scale)
		radius := math.Abs(local.Radius) * math.Max(math.Abs(scale.X), math.Abs(scale.Y))
		world := geom.Circle{Center: center, Radius: radius}
		return transformedShape{shape: world, bounds: world.BoundingBox()}, nil
	default:
		return transformedShape{}, fmt.Errorf("unsupported collider shape %T", shape)
	}
}

func transformPoints(points [4]geom.Vector2, transform components.Transform) [4]geom.Vector2 {
	var transformed [4]geom.Vector2
	for i, point := range points {
		transformed[i] = transformPoint(point, transform)
	}
	return transformed
}

func transformPoint(point geom.Vector2, transform components.Transform) geom.Vector2 {
	scale := effectiveScale(transform.Scale)
	scaled := geom.Vector2{X: point.X * scale.X, Y: point.Y * scale.Y}
	return scaled.Rotate(transform.Rotation).Add(transform.Position)
}

func effectiveScale(scale geom.Vector2) geom.Vector2 {
	// Zero-value transforms are common in existing callers, so zero scale keeps
	// its historical meaning of identity rather than collapsing the collider.
	if scale.X == 0 {
		scale.X = 1
	}
	if scale.Y == 0 {
		scale.Y = 1
	}
	return scale
}

func boundsOf(points []geom.Vector2) geom.Rect {
	min := points[0]
	max := points[0]
	for _, point := range points[1:] {
		min.X = math.Min(min.X, point.X)
		min.Y = math.Min(min.Y, point.Y)
		max.X = math.Max(max.X, point.X)
		max.Y = math.Max(max.Y, point.Y)
	}
	return geom.Rect{Min: min, Max: max}
}

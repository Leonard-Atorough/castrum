// Package collision provides collision detection for entities: the
// [Collider] component with its sealed [ColliderShape] sum, the
// rotation-aware narrow phase for shape pairs, and the system that
// records each tick's results as [Contacts] state on the collider
// entities.
package collision

import (
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// worldShape is a collider's geometry after the transform applies:
// rotation and position are baked in, so narrow-phase tests work in
// world space directly. A sealed sum, like [ColliderShape], so the
// pair dispatch is one type switch.
type worldShape interface {
	isWorldShape()
}

// orientedRect is a rectangle rotated into world space: its center,
// its half-extents before rotation, and its two orthonormal world
// axes.
type orientedRect struct {
	center      geom.Vector2
	halfExtents geom.Vector2
	axes        [2]geom.Vector2
}

func (orientedRect) isWorldShape() {}

// worldCircle is a circle in world space.
type worldCircle struct {
	center geom.Vector2
	radius float64
}

func (worldCircle) isWorldShape() {}

// contact is the exact narrow-phase result for a shape pair:
// collided, the shared contact point, the contact normal, and the
// overlap depth along it. Normal points from the first shape toward
// the second.
type contact struct {
	collided    bool
	point       geom.Vector2
	normal      geom.Vector2
	penetration float64
}

// transformedShape keeps the exact world-space shape alongside its
// conservative world-space AABB. The exact shape is for narrow-phase
// tests; the AABB is for broad-phase indexing.
type transformedShape struct {
	shape  worldShape
	bounds geom.Rect
}

// transformShape applies the collider's local-space offset, then the
// transform's rotation and position. Scale is ignored: transform
// scale is presentation state - squash-and-stretch, death effects -
// and non-uniform scale would turn circles into ellipses the narrow
// phase cannot test. A game that wants a hitbox to follow scale
// writes the shape explicitly.
//
// The zero value is returned for a shape outside the sum; storage
// validation keeps that from ever reaching the system.
func transformShape(shape ColliderShape, offset geom.Vector2, transform core.Transform) transformedShape {
	switch local := shape.(type) {
	case RectShape:
		ox, oy := offset.X, offset.Y
		corners := [4]geom.Vector2{
			{X: local.Min.X + ox, Y: local.Min.Y + oy},
			{X: local.Max.X + ox, Y: local.Min.Y + oy},
			{X: local.Max.X + ox, Y: local.Max.Y + oy},
			{X: local.Min.X + ox, Y: local.Max.Y + oy},
		}
		world := transformCorners(corners, transform)
		center := transformPoint(geom.Vector2{
			X: (local.Min.X+local.Max.X)/2 + ox,
			Y: (local.Min.Y+local.Max.Y)/2 + oy,
		}, transform)
		return transformedShape{
			shape: orientedRect{
				center:      center,
				halfExtents: geom.Vector2{X: math.Abs(local.Max.X-local.Min.X) / 2, Y: math.Abs(local.Max.Y-local.Min.Y) / 2},
				axes: [2]geom.Vector2{
					{X: math.Cos(transform.Rotation), Y: math.Sin(transform.Rotation)},
					{X: -math.Sin(transform.Rotation), Y: math.Cos(transform.Rotation)},
				},
			},
			bounds: boundsOf(world[:]),
		}
	case CircleShape:
		center := transformPoint(geom.Vector2{
			X: local.Center.X + offset.X,
			Y: local.Center.Y + offset.Y,
		}, transform)
		return transformedShape{
			shape:  worldCircle{center: center, radius: local.Radius},
			bounds: geom.RectFromCenterSize(center, geom.Vector2{X: local.Radius * 2, Y: local.Radius * 2}),
		}
	default:
		return transformedShape{}
	}
}

// transformPoint applies only rotation and position, ignoring scale.
func transformPoint(point geom.Vector2, transform core.Transform) geom.Vector2 {
	return point.Rotate(transform.Rotation).Add(transform.Position)
}

func transformCorners(points [4]geom.Vector2, transform core.Transform) [4]geom.Vector2 {
	var transformed [4]geom.Vector2
	for i, point := range points {
		transformed[i] = transformPoint(point, transform)
	}
	return transformed
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

// contactBetween runs the exact narrow-phase test for a shape pair.
// Touching counts: a pair sharing only a boundary point collides.
// Normal always points from the first shape toward the second.
func contactBetween(a, b worldShape) contact {
	switch a := a.(type) {
	case orientedRect:
		switch b := b.(type) {
		case orientedRect:
			return rectRectContact(a, b)
		case worldCircle:
			// circleRectContact reports circle toward rect; the pair
			// here is rect toward circle.
			hit := circleRectContact(b, a)
			hit.normal = hit.normal.Mul(-1)
			return hit
		}
	case worldCircle:
		switch b := b.(type) {
		case orientedRect:
			return circleRectContact(a, b)
		case worldCircle:
			return circleCircleContact(a, b)
		}
	}
	return contact{}
}

func circleCircleContact(a, b worldCircle) contact {
	delta := b.center.Sub(a.center)
	dist := delta.Length()
	minDist := a.radius + b.radius
	if dist > minDist {
		return contact{}
	}

	if dist == 0 {
		// Coincident centers have no unique normal. Use a stable fallback.
		return contact{
			collided:    true,
			penetration: minDist,
			normal:      geom.Vector2{X: 1, Y: 0},
			point:       a.center,
		}
	}

	normal := delta.Div(dist)
	return contact{
		collided:    true,
		penetration: minDist - dist,
		normal:      normal,
		point:       a.center.Add(normal.Mul(a.radius)),
	}
}

func circleRectContact(circle worldCircle, rect orientedRect) contact {
	local := circle.center.Sub(rect.center)
	projectionX := local.Dot(rect.axes[0])
	projectionY := local.Dot(rect.axes[1])
	closestLocal := geom.Vector2{
		X: math.Max(-rect.halfExtents.X, math.Min(projectionX, rect.halfExtents.X)),
		Y: math.Max(-rect.halfExtents.Y, math.Min(projectionY, rect.halfExtents.Y)),
	}
	closest := rect.center.
		Add(rect.axes[0].Mul(closestLocal.X)).
		Add(rect.axes[1].Mul(closestLocal.Y))

	delta := circle.center.Sub(closest)
	dist := delta.Length()
	if dist > circle.radius {
		return contact{}
	}

	if dist == 0 {
		// The closest point is the circle center when it is inside the
		// rectangle, so there is no unique contact normal. Use a stable
		// fallback direction.
		return contact{
			collided:    true,
			penetration: circle.radius,
			normal:      geom.Vector2{X: 1, Y: 0},
			point:       circle.center,
		}
	}

	normal := closest.Sub(circle.center).Div(dist)
	return contact{
		collided:    true,
		penetration: circle.radius - dist,
		normal:      normal,
		point:       closest,
	}
}

func rectRectContact(a, b orientedRect) contact {
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
			return contact{}
		}
		if overlap < minimumOverlap {
			minimumOverlap = overlap
			minimumAxis = axis
		}
	}

	if centerDelta.Dot(minimumAxis) < 0 {
		minimumAxis = minimumAxis.Mul(-1)
	}
	point := a.center.
		Add(a.axes[0].Mul(a.halfExtents.X * math.Copysign(1, minimumAxis.Dot(a.axes[0])))).
		Add(a.axes[1].Mul(a.halfExtents.Y * math.Copysign(1, minimumAxis.Dot(a.axes[1]))))
	return contact{
		collided:    true,
		penetration: minimumOverlap,
		normal:      minimumAxis,
		point:       point,
	}
}

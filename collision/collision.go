package collision

import (
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

type worldShape interface {
	isWorldShape()
}

type orientedRect struct {
	center      geom.Vector2
	halfExtents geom.Vector2
	axes        [2]geom.Vector2
}

func (orientedRect) isWorldShape() {}

type worldCircle struct {
	center geom.Vector2
	radius float64
}

func (worldCircle) isWorldShape() {}

// bounds must enclose shape so broad-phase queries cannot miss overlaps.
type transformedShape struct {
	shape  worldShape
	bounds geom.Rect
}

// transformShape applies the collider's local-space offset, then the
// transform's rotation and position. Scale is intentionally ignored.
func transformShape(shape ColliderShape, offset geom.Vector2, transform core.Transform) transformedShape {
	switch local := shape.(type) {
	case Box:
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
	case Circle:
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

type contact struct {
	collided    bool
	point       geom.Vector2
	normal      geom.Vector2
	penetration float64
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
			// When the circle is the second shape and the rectangle is the first,
			// we compute the contact normally and then invert the normal to point
			// from the rectangle to the circle.
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

	// Test containment in box space; reconstructing closest in world space
	// can introduce a nonzero distance for a contained circle in a rotated box.
	if closestLocal.X == projectionX && closestLocal.Y == projectionY {
		depthX := rect.halfExtents.X - math.Abs(projectionX)
		depthY := rect.halfExtents.Y - math.Abs(projectionY)
		var normal, depth = geom.Vector2{}, 0.0
		if depthX <= depthY {
			depth, normal = depthX, rect.axes[0].Mul(math.Copysign(1, projectionX))
		} else {
			depth, normal = depthY, rect.axes[1].Mul(math.Copysign(1, projectionY))
		}
		return contact{
			collided:    true,
			penetration: depth + circle.radius,
			normal:      normal.Mul(-1),
			point:       circle.center.Add(normal.Mul(depth)),
		}
	}

	delta := circle.center.Sub(closest)
	dist := delta.Length()
	if dist > circle.radius {
		return contact{}
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

type segmentHit struct {
	hit    bool
	t      float64
	point  geom.Vector2
	normal geom.Vector2
}

func segmentShapeHit(segment geom.Segment, shape worldShape) segmentHit {
	switch s := shape.(type) {
	case worldCircle:
		return segmentCircleHit(segment, s)
	case orientedRect:
		return segmentRectHit(segment, s)
	}
	return segmentHit{}
}

func radialNormal(offset geom.Vector2) geom.Vector2 {
	if offset.X == 0 && offset.Y == 0 {
		return geom.Vector2{X: 1, Y: 0}
	}
	return offset.Normalize()
}

func segmentCircleHit(segment geom.Segment, circle worldCircle) segmentHit {
	direction := segment.End.Sub(segment.Start)
	offset := segment.Start.Sub(circle.center)
	if offset.Length() <= circle.radius {
		return segmentHit{hit: true, point: segment.Start, normal: radialNormal(offset)}
	}
	length := direction.Length()
	if length == 0 {
		return segmentHit{}
	}
	unit := direction.Normalize()
	along := -offset.Dot(unit)
	perpendicular := offset.Cross(unit)
	if along < 0 || math.Abs(perpendicular) > circle.radius {
		return segmentHit{}
	}
	// Closest approach avoids subtracting nearly equal quadratic terms.
	ratio := math.Abs(perpendicular) / circle.radius
	halfChord := circle.radius * math.Sqrt((1-ratio)*(1+ratio))
	distance := along - halfChord
	if distance < 0 || distance > length {
		return segmentHit{}
	}
	point := segment.Start.Add(unit.Mul(distance))
	normal := geom.Vector2{X: unit.Y * perpendicular, Y: -unit.X * perpendicular}.
		Sub(unit.Mul(halfChord)).Normalize()
	return segmentHit{
		hit:    true,
		t:      distance / length,
		point:  point,
		normal: normal,
	}
}

func segmentRectHit(segment geom.Segment, rect orientedRect) segmentHit {
	localStart := segment.Start.Sub(rect.center)
	localEnd := segment.End.Sub(rect.center)
	origins := [2]float64{localStart.Dot(rect.axes[0]), localStart.Dot(rect.axes[1])}
	directions := [2]float64{
		localEnd.Dot(rect.axes[0]) - origins[0],
		localEnd.Dot(rect.axes[1]) - origins[1],
	}
	extents := [2]float64{rect.halfExtents.X, rect.halfExtents.Y}

	tEnter, tExit := math.Inf(-1), math.Inf(1)
	var normal geom.Vector2
	for axis := range origins {
		if directions[axis] == 0 {
			if math.Abs(origins[axis]) > extents[axis] {
				return segmentHit{}
			}
			continue
		}

		t1 := (-extents[axis] - origins[axis]) / directions[axis]
		t2 := (extents[axis] - origins[axis]) / directions[axis]
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tEnter {
			tEnter = t1
			normal = rect.axes[axis]
			if directions[axis] > 0 {
				normal = normal.Mul(-1)
			}
		}
		tExit = math.Min(tExit, t2)
		if tEnter > tExit {
			return segmentHit{}
		}
	}

	if tExit < 0 || tEnter > 1 {
		return segmentHit{}
	}

	t := tEnter
	if t < 0 {
		t = 0
		projectionX, projectionY := origins[0], origins[1]
		depthX := rect.halfExtents.X - math.Abs(projectionX)
		depthY := rect.halfExtents.Y - math.Abs(projectionY)
		if depthX <= depthY {
			normal = rect.axes[0].Mul(math.Copysign(1, projectionX))
		} else {
			normal = rect.axes[1].Mul(math.Copysign(1, projectionY))
		}
	}

	direction := segment.End.Sub(segment.Start)
	return segmentHit{
		hit:    true,
		t:      t,
		point:  segment.Start.Add(direction.Mul(t)),
		normal: normal,
	}
}

// Package physicssystem processes collision detection and resolution for
// entities with Collider components using spatial indexing for efficient queries.
package physics

import (
	"fmt"
	"math"
	"strings"

	"github.com/leonard-atorough/castrum/geom"
)

type CollisionState struct {
	CollisionResult
	WasColliding bool
}

// CollisionErrors accumulates errors encountered during collision testing.
// This allows bulk error reporting without I/O overhead per error.
type CollisionErrors struct {
	Errors []error
}

// Add appends an error to the collection.
func (ce *CollisionErrors) Add(err error) {
	if err != nil {
		ce.Errors = append(ce.Errors, err)
	}
}

// Error returns a formatted string of all accumulated errors.
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

// Unwrap returns the slice of accumulated errors for inspection with errors.Is/As.
func (ce *CollisionErrors) Unwrap() []error {
	return ce.Errors
}

// IsEmpty returns true if no errors have been accumulated.
func (ce *CollisionErrors) IsEmpty() bool {
	return len(ce.Errors) == 0
}

// toWorldSpace translates a collider shape to world space using the entity's transform position
func toWorldSpace(shape any, position geom.Vector2) any {
	switch s := shape.(type) {
	case geom.Rect:
		return geom.Rect{
			Min: geom.Vector2{X: s.Min.X + position.X, Y: s.Min.Y + position.Y},
			Max: geom.Vector2{X: s.Max.X + position.X, Y: s.Max.Y + position.Y},
		}
	case geom.Circle:
		return geom.Circle{
			Center: geom.Vector2{X: s.Center.X + position.X, Y: s.Center.Y + position.Y},
			Radius: s.Radius,
		}
	}
	return shape
}

func intersectsAny(shapeA, shapeB any) CollisionResult {
	switch a := shapeA.(type) {
	case geom.Rect:
		switch b := shapeB.(type) {
		case geom.Rect:
			// Rect-Rect contact geometry deferred (SAT solver v0.2.0)
			return CollisionResult{Collided: a.Intersects(b)}
		case geom.Circle:
			return circleRectContact(b, a)
		}
	case geom.Circle:
		switch b := shapeB.(type) {
		case geom.Rect:
			return circleRectContact(a, b)
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
		// Circles at same position, arbitrary normal.
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

func circleRectContact(circle geom.Circle, rect geom.Rect) CollisionResult {
	// Closest point on rect to circle center.
	closestX := math.Max(rect.Min.X, math.Min(circle.Center.X, rect.Max.X))
	closestY := math.Max(rect.Min.Y, math.Min(circle.Center.Y, rect.Max.Y))

	dx := circle.Center.X - closestX
	dy := circle.Center.Y - closestY
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > circle.Radius {
		return CollisionResult{Collided: false}
	}

	if dist == 0 {
		// Circle center inside rect, arbitrary normal outward.
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
		Point:       geom.Vector2{X: closestX, Y: closestY},
	}
}

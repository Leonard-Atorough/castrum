package geom

import "math"

type CollisionResult struct {
	Collided    bool
	Normal      Vector2
	Penetration float64
}

func IntersectsAny(shapeA, shapeB any) CollisionResult {
	switch a := shapeA.(type) {
	case Rect:
		switch b := shapeB.(type) {
		case Rect:
			// Implement Rect-Rect collision detection
			return CollisionResult{Collided: a.Intersects(b)}
		case Circle:
			// Implement Rect-Circle collision detection
			return CollisionResult{Collided: CircleIntersectsRect(b, a)}
		}
	case Circle:
		switch b := shapeB.(type) {
		case Rect:
			// Implement Circle-Rect collision detection
			return CollisionResult{Collided: CircleIntersectsRect(a, b)}
		case Circle:
			// Implement Circle-Circle collision detection
			return CollisionResult{Collided: a.Intersects(b)}
		}
	}
	return CollisionResult{}
}

func CircleIntersectsRect(circle Circle, rect Rect) bool {
	dx := circle.Center.X - math.Max(rect.Min.X, math.Min(circle.Center.X, rect.Max.X))
	dy := circle.Center.Y - math.Max(rect.Min.Y, math.Min(circle.Center.Y, rect.Max.Y))
	// Calculate the distance from the circle's center to the closest point on the rectangle's edge
	return dx*dx+dy*dy <= circle.Radius*circle.Radius
}

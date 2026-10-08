package collision

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Contact is one detected overlap from this collider's point of view.
// The two colliders' records share the point and penetration, while
// their normals point in opposite directions. Point is the
// narrow-phase test's representative contact point; it is not a
// contact manifold.
type Contact struct {
	// Other is the entity this contact is with.
	Other core.EntityID
	// Point is the representative contact point returned by the
	// narrow-phase test for this pair.
	Point geom.Vector2
	// Normal points from this collider toward Other.
	Normal geom.Vector2
	// Penetration is the overlap depth along the normal. Touching
	// shapes have zero penetration and still produce a contact.
	Penetration float64
	// Trigger reports whether either collider in the pair is a
	// trigger. A game reacting to trigger contacts reads this instead
	// of fetching the other collider.
	Trigger bool
}

// Contacts is the collision lifecycle state of one collider entity,
// written by the system from [NewSystem] every fixed tick. Current
// holds this tick's contacts and Previous the last tick's, so the
// edges are differences:
//
//	enter: in Current, not in Previous
//	stay:  in both
//	exit:  in Previous, not in Current
//
// Both slices are sorted by [Contact.Other]. A game derives its own
// enter/stay/exit reactions by comparing the two; the engine emits
// no events.
//
// The collision system attaches Contacts to every entity that
// carries a Collider, and it alone writes it.
type Contacts struct {
	// Current is this tick's contacts, sorted by Other.
	Current []Contact
	// Previous is the contacts of the tick before this one, sorted
	// by Other.
	Previous []Contact
}

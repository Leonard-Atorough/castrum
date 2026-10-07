package collision

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Contact is one colliding pair from this collider's point of view.
// The two colliders' records for a pair mirror each other: same
// point and penetration, normals pointing toward the other side.
type Contact struct {
	// Other is the entity this contact is with.
	Other core.EntityID
	// Point is the pair's contact point.
	Point geom.Vector2
	// Normal points from this collider toward Other.
	Normal geom.Vector2
	// Penetration is how far the two shapes overlap along the normal.
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
// carries a Collider, and it alone writes it. Game code reads it but
// never writes or removes it - the same ownership rule as
// [core.PrevTransform].
type Contacts struct {
	// Current is this tick's contacts, sorted by Other.
	Current []Contact
	// Previous is the contacts of the tick before this one, sorted
	// by Other.
	Previous []Contact
}

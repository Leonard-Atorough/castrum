package collision

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Contact describes one detected overlap between this collider and [Contact.Other].
//
// Each collider has its own record for a pair. Both records share the
// point and penetration, but their normals point in opposite directions.
// Point is representative of the overlap, not a contact manifold.
type Contact struct {
	// Other is the entity this contact is with.
	Other core.EntityID
	// Point is the pair's representative contact point.
	Point geom.Vector2
	// Normal points from this collider toward Other.
	Normal geom.Vector2
	// Penetration is the overlap depth along the normal. Touching
	// shapes have zero penetration and still produce a contact.
	Penetration float64
	// Trigger is true if either collider in the pair is a trigger.
	Trigger bool
}

// Contacts is the collision system's per-entity snapshot of detected overlaps.
//
// [NewSystem] writes it for entities with both a [Collider] and a
// [core.Transform] during fixed updates. [Contacts.Current] contains the
// contacts detected in the latest update; [Contacts.Previous] contains
// those from the update before it. Comparing their [Contact.Other] IDs
// identifies contacts that began, continued, or ended:
//
//	enter: in Current, not in Previous
//	stay:  in both
//	exit:  in Previous, not in Current
//
// Both slices are sorted by [Contact.Other]. For enter and exit edges,
// games can also consume [Entered] and [Exited]; Contacts remains the
// source of truth for current contacts and stay state. The slices are
// read-only and valid until the next collision update. Copy them to keep
// contact history across updates.
//
// The collision system attaches and owns Contacts, removing it when
// either [Collider] or [core.Transform] is removed. Game code reads it
// but does not write or remove it.
type Contacts struct {
	// Current is this tick's contacts, sorted by Other.
	Current []Contact
	// Previous is the contacts of the tick before this one, sorted
	// by Other.
	Previous []Contact
}

// Entered is emitted when this collider's [Contacts.Current] gains a
// contact during a fixed update.
//
// The collision system emits one signal per perspective, so a pair
// produces two signals with opposing normals. Contact geometry matches
// the new record in [Contacts.Current].
type Entered struct {
	// Entity is the collider whose Contacts gained the pair.
	Entity core.EntityID
	// Other is the entity it began overlapping.
	Other core.EntityID
	// Point is the pair's representative contact point.
	Point geom.Vector2
	// Normal points from Entity toward Other.
	Normal geom.Vector2
	// Penetration is the overlap depth along the normal.
	Penetration float64
	// Trigger is true if either collider in the pair is a trigger.
	Trigger bool
	// Tick is the fixed tick the pair began overlapping on.
	Tick uint64
}

// Exited is emitted when this collider's [Contacts.Current] loses a
// contact during a fixed update.
//
// The collision system emits one signal per perspective, like [Entered].
// Since the pair no longer has contact geometry, the signal carries only
// the entity IDs and tick. A collider removed during the update does not
// emit its own exits; any remaining collider's signal reports the pair.
type Exited struct {
	// Entity is the collider whose Contacts lost the pair.
	Entity core.EntityID
	// Other is the entity it stopped overlapping.
	Other core.EntityID
	// Tick is the fixed tick the pair ended on.
	Tick uint64
}

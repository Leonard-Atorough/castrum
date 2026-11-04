// Package collision detects overlaps between entity colliders and records
// them in [Contacts]. [Collider] supports local-space [Box] and [Circle]
// shapes, transformed by an entity's position and rotation (scale is ignored).
// Create colliders with [NewCollider], attach them to entities with a
// [core.Transform], and register [NewSystem] in the fixed schedule. Gameplay
// reads the enter, stay, and exit lifecycle from [Contacts.Current] and
// [Contacts.Previous], or reacts to the [Entered] and [Exited] signals the
// system emits as pairs begin and end; the package does not resolve
// collisions. The system also publishes a [Queries] resource for raycasts
// against the same synchronized collider state.
package collision

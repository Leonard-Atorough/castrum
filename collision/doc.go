// Package collision detects overlaps between entity colliders and records
// them in [Contacts]. [Collider] supports local-space [Box] and [Circle]
// shapes, transformed by entity position and rotation (scale is ignored).
// Create colliders with [NewCollider], attach them to entities with a
// [core.Transform], and register [NewSystem] in the fixed schedule. Read
// overlap lifecycle from [Contacts] or listen for [Entered] and [Exited].
// The package detects collisions but does not resolve them. The collision
// system updates [Queries] for raycasts against the same synchronized
// collider state.
package collision

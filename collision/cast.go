package collision

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

type RaycastFilter struct {
	// Mask specifies the collision mask to use for the raycast. Only colliders
	// matching this mask will be considered.
	Mask uint32
	// Exclude specifies a list of entity IDs to ignore during the raycast.
	Exclude []core.EntityID
}

type RaycastHit struct {
	Entity   core.EntityID
	Point    geom.Vector2
	Normal   geom.Vector2
	Distance float64
}

// Raycast performs a raycast in the given world along the specified segment,
// applying the provided filter. It returns the first hit encountered, a boolean
// indicating if a hit occurred, and an error if the operation failed.
func Raycast(world *core.World, segment geom.Segment, filter RaycastFilter) (RaycastHit, bool, error) {
	return RaycastHit{}, false, nil
}

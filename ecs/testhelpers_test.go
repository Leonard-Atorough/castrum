package ecs

// idsMatchUnordered checks if two entity ID slices contain the same elements regardless of order.
func idsMatchUnordered(got, want []EntityID) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[EntityID]bool)
	for _, id := range want {
		seen[id] = true
	}
	for _, id := range got {
		if !seen[id] {
			return false
		}
	}
	return true
}

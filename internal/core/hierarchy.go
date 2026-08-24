package core

import "github.com/leonard-atorough/castrum/ecs"

// Hierarchy models a parent/child relationship between entities.
// parentToChildren maps a parent entity ID to its children.
// childToParent maps a child entity ID to its parent.
type Hierarchy struct {
	parentToChildren map[ecs.EntityID][]ecs.EntityID
	childToParent    map[ecs.EntityID]ecs.EntityID
}

func NewHierarchy() *Hierarchy {
	return &Hierarchy{
		parentToChildren: make(map[ecs.EntityID][]ecs.EntityID),
		childToParent:    make(map[ecs.EntityID]ecs.EntityID),
	}
}

// Add attaches childID under parentID.
// If childID already has a parent, it is first detached from that parent.
func (h *Hierarchy) Add(parentID, childID ecs.EntityID) {
	if oldParent, exists := h.childToParent[childID]; exists && oldParent != parentID {
		h.Remove(oldParent, childID)
	}

	if _, exists := h.parentToChildren[parentID]; !exists {
		h.parentToChildren[parentID] = nil
	}

	for _, id := range h.parentToChildren[parentID] {
		if id == childID {
			return
		}
	}

	h.parentToChildren[parentID] = append(h.parentToChildren[parentID], childID)
	h.childToParent[childID] = parentID
}

// Remove detaches childID from parentID.
func (h *Hierarchy) Remove(parentID, childID ecs.EntityID) {
	children := h.parentToChildren[parentID]
	for i, id := range children {
		if id == childID {
			h.parentToChildren[parentID] = append(children[:i], children[i+1:]...)
			delete(h.childToParent, childID)
			if len(h.parentToChildren[parentID]) == 0 {
				delete(h.parentToChildren, parentID)
			}
			return
		}
	}
}

// Children returns all direct children of parentID.
func (h *Hierarchy) Children(parentID ecs.EntityID) []ecs.EntityID {
	return append([]ecs.EntityID(nil), h.parentToChildren[parentID]...)
}

// Parent returns the parent of childID, if any.
func (h *Hierarchy) Parent(childID ecs.EntityID) (ecs.EntityID, bool) {
	parent, ok := h.childToParent[childID]
	return parent, ok
}

// IsParent returns true if entityID has any children.
func (h *Hierarchy) IsParent(entityID ecs.EntityID) bool {
	_, exists := h.parentToChildren[entityID]
	return exists
}

// IsChild returns true if entityID has a parent.
func (h *Hierarchy) IsChild(entityID ecs.EntityID) bool {
	_, exists := h.childToParent[entityID]
	return exists
}

// Descendants returns all descendants of parentID in depth-first order.
func (h *Hierarchy) Descendants(parentID ecs.EntityID) []ecs.EntityID {
	result := make([]ecs.EntityID, 0)
	var walk func(ecs.EntityID)
	walk = func(id ecs.EntityID) {
		for _, childID := range h.parentToChildren[id] {
			result = append(result, childID)
			walk(childID)
		}
	}
	walk(parentID)
	return result
}

// Root walks up the chain and returns the root ancestor of entityID.
// If entityID is not attached to a parent, it returns entityID itself.
func (h *Hierarchy) Root(entityID ecs.EntityID) ecs.EntityID {
	for {
		parent, ok := h.childToParent[entityID]
		if !ok {
			return entityID
		}
		entityID = parent
	}
}

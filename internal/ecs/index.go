package ecs

import (
	"reflect"
	"slices"
)

// entityIndex maintains an index of components, tags, and templates for efficient querying.
type entityIndex struct {
	// Component index maps component names to a set of entity IDs that have that component.
	// The map structure is: componentName -> list of entityIDs
	componentIndex map[reflect.Type][]uint64

	// Tag index maps tag names to a set of entity IDs that have that tag.
	// The map structure is: tagName -> list of entityIDs
	tagIndex map[string][]uint64

	// Template index maps template names to a set of entity IDs that use that template.
	// The map structure is: templateName -> list of entityIDs
	templateIndex map[string][]uint64
}

func NewEntityIndex() entityIndex {
	return entityIndex{
		componentIndex: make(map[reflect.Type][]uint64),
		tagIndex:       make(map[string][]uint64),
		templateIndex:  make(map[string][]uint64),
	}
}

func (idx *entityIndex) AddComponent(entityID uint64, compType reflect.Type) {
	idx.addToIndex(idx.componentIndex, compType, entityID)
}

func (idx *entityIndex) RemoveComponent(entityID uint64, compType reflect.Type) {
	idx.removeFromIndex(idx.componentIndex, compType, entityID)
}

func (idx *entityIndex) AddTag(entityID uint64, tagName string) {
	idx.addToIndexString(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) RemoveTag(entityID uint64, tagName string) {
	idx.removeFromIndexString(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) AddTemplate(entityID uint64, templateName string) {
	idx.addToIndexString(idx.templateIndex, templateName, entityID)
}

func (idx *entityIndex) RemoveTemplate(entityID uint64, templateName string) {
	idx.removeFromIndexString(idx.templateIndex, templateName, entityID)
}

func (idx *entityIndex) UpdateComponent(entity *entity, componentType reflect.Type, add bool) {
	if add {
		idx.addToIndex(idx.componentIndex, componentType, entity.id)
	} else {
		idx.removeFromIndex(idx.componentIndex, componentType, entity.id)
	}
}

func (idx *entityIndex) UpdateTag(entity *entity, tagName string, add bool) {
	if add {
		idx.addToIndexString(idx.tagIndex, tagName, entity.id)
	} else {
		idx.removeFromIndexString(idx.tagIndex, tagName, entity.id)
	}
}

func (idx *entityIndex) UpdateTemplate(entity *entity, oldTemplate string, newTemplate string) {
	if oldTemplate != "" {
		idx.removeFromIndexString(idx.templateIndex, oldTemplate, entity.id)
	}
	if newTemplate != "" {
		idx.addToIndexString(idx.templateIndex, newTemplate, entity.id)
	}
}

func (idx *entityIndex) GetEntitiesWithComponent(componentType reflect.Type) []uint64 {
	return idx.getFromIndex(idx.componentIndex, componentType)
}

func (idx *entityIndex) GetEntitiesWithComponents(componentTypes ...reflect.Type) []uint64 {
	if len(componentTypes) == 0 {
		return nil
	}

	resultSet := make(map[uint64]bool)
	for _, ct := range componentTypes {
		currentList := idx.GetEntitiesWithComponent(ct)
		if len(currentList) == 0 {
			return nil
		}

		if len(resultSet) == 0 {
			for _, id := range currentList {
				resultSet[id] = true
			}
			continue
		}

		nextSet := make(map[uint64]bool, len(currentList))
		for _, id := range currentList {
			nextSet[id] = true
		}

		for id := range resultSet {
			if !nextSet[id] {
				delete(resultSet, id)
			}
		}
	}

	out := make([]uint64, 0, len(resultSet))
	for id := range resultSet {
		out = append(out, id)
	}
	return out
}

func (idx *entityIndex) GetEntitiesWithTag(tagName string) []uint64 {
	return idx.getFromIndexString(idx.tagIndex, tagName)
}

func (idx *entityIndex) GetEntitiesWithTemplate(templateName string) []uint64 {
	return idx.getFromIndexString(idx.templateIndex, templateName)
}

func (idx *entityIndex) addToIndex(index map[reflect.Type][]uint64, key reflect.Type, entityID uint64) {
	if _, exists := index[key]; !exists {
		index[key] = make([]uint64, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) addToIndexString(index map[string][]uint64, key string, entityID uint64) {
	if _, exists := index[key]; !exists {
		index[key] = make([]uint64, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) removeFromIndex(index map[reflect.Type][]uint64, key reflect.Type, entityID uint64) {
	index[key] = removeEntityID(index[key], entityID)
}

func (idx *entityIndex) removeFromIndexString(index map[string][]uint64, key string, entityID uint64) {
	index[key] = removeEntityID(index[key], entityID)
}

func removeEntityID(entities []uint64, entityID uint64) []uint64 {
	for i, id := range entities {
		if id == entityID {
			entities = append(entities[:i], entities[i+1:]...)
			break
		}
	}
	if len(entities) == 0 {
		entities = nil
	}
	return entities
}

func (idx *entityIndex) getFromIndex(index map[reflect.Type][]uint64, key reflect.Type) []uint64 {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

func (idx *entityIndex) getFromIndexString(index map[string][]uint64, key string) []uint64 {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

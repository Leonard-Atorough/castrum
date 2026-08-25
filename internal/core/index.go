package core

import (
	"reflect"
	"slices"

	"github.com/leonard-atorough/castrum/ecs"
)

// entityIndex maintains an index of components, tags, and templates for efficient querying.
type entityIndex struct {
	// Component index maps component names to a set of entity IDs that have that component.
	// The map structure is: componentName -> list of entityIDs
	componentIndex map[reflect.Type][]ecs.EntityID

	// Tag index maps tag names to a set of entity IDs that have that tag.
	// The map structure is: tagName -> list of entityIDs
	tagIndex map[string][]ecs.EntityID

	// Template index maps template names to a set of entity IDs that use that template.
	// The map structure is: templateName -> list of entityIDs
	templateIndex map[string][]ecs.EntityID

	// lazyTagTemplateIndex indicates whether the tag/template indexes are stale because
	// entities were created without eagerly updating lookup metadata.
	lazyTagTemplateIndex bool
}

func NewEntityIndex() entityIndex {
	return entityIndex{
		componentIndex:       make(map[reflect.Type][]ecs.EntityID),
		tagIndex:             make(map[string][]ecs.EntityID),
		templateIndex:        make(map[string][]ecs.EntityID),
		lazyTagTemplateIndex: false,
	}
}

func (idx *entityIndex) AddComponent(entityID ecs.EntityID, compType reflect.Type) {
	idx.addToIndex(idx.componentIndex, compType, entityID)
}

func (idx *entityIndex) RemoveComponent(entityID ecs.EntityID, compType reflect.Type) {
	idx.removeFromIndex(idx.componentIndex, compType, entityID)
}

func (idx *entityIndex) AddTag(entityID ecs.EntityID, tagName string) {
	idx.addToIndexString(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) RemoveTag(entityID ecs.EntityID, tagName string) {
	idx.removeFromIndexString(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) AddTemplate(entityID ecs.EntityID, templateName string) {
	idx.addToIndexString(idx.templateIndex, templateName, entityID)
}

func (idx *entityIndex) RemoveTemplate(entityID ecs.EntityID, templateName string) {
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

func (idx *entityIndex) GetEntitiesWithComponent(componentType reflect.Type) []ecs.EntityID {
	return idx.getFromIndex(idx.componentIndex, componentType)
}

func (idx *entityIndex) GetEntitiesWithComponents(componentTypes ...reflect.Type) []ecs.EntityID {
	if len(componentTypes) == 0 {
		return nil
	}

	resultSet := make(map[ecs.EntityID]bool)
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

		nextSet := make(map[ecs.EntityID]bool, len(currentList))
		for _, id := range currentList {
			nextSet[id] = true
		}

		for id := range resultSet {
			if !nextSet[id] {
				delete(resultSet, id)
			}
		}
	}

	out := make([]ecs.EntityID, 0, len(resultSet))
	for id := range resultSet {
		out = append(out, id)
	}
	return out
}

func (idx *entityIndex) GetEntitiesWithTag(tagName string) []ecs.EntityID {
	return idx.getFromIndexString(idx.tagIndex, tagName)
}

func (idx *entityIndex) GetEntitiesWithTemplate(templateName string) []ecs.EntityID {
	return idx.getFromIndexString(idx.templateIndex, templateName)
}

func (idx *entityIndex) addToIndex(index map[reflect.Type][]ecs.EntityID, key reflect.Type, entityID ecs.EntityID) {
	if _, exists := index[key]; !exists {
		index[key] = make([]ecs.EntityID, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) addToIndexString(index map[string][]ecs.EntityID, key string, entityID ecs.EntityID) {
	if _, exists := index[key]; !exists {
		index[key] = make([]ecs.EntityID, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) rebuildTagAndTemplateIndex(entities map[ecs.EntityID]*entity) {
	idx.tagIndex = make(map[string][]ecs.EntityID)
	idx.templateIndex = make(map[string][]ecs.EntityID)

	for entityID, entity := range entities {
		if entity == nil {
			continue
		}
		if entity.template != "" {
			idx.addToIndexString(idx.templateIndex, entity.template, entityID)
		}
		for tag := range entity.tags {
			idx.addToIndexString(idx.tagIndex, tag, entityID)
		}
	}
	idx.lazyTagTemplateIndex = false
}

func (idx *entityIndex) removeFromIndex(index map[reflect.Type][]ecs.EntityID, key reflect.Type, entityID ecs.EntityID) {
	index[key] = removeEntityID(index[key], entityID)
}

func (idx *entityIndex) removeFromIndexString(index map[string][]ecs.EntityID, key string, entityID ecs.EntityID) {
	index[key] = removeEntityID(index[key], entityID)
}

func removeEntityID(entities []ecs.EntityID, entityID ecs.EntityID) []ecs.EntityID {
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

func (idx *entityIndex) getFromIndex(index map[reflect.Type][]ecs.EntityID, key reflect.Type) []ecs.EntityID {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

func (idx *entityIndex) getFromIndexString(index map[string][]ecs.EntityID, key string) []ecs.EntityID {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

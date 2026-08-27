package core

import (
	"reflect"
	"slices"
)

// entityIndex maintains an index of components, tags, and templates for efficient querying.
type entityIndex struct {

	// Tag index maps tag names to a set of entity IDs that have that tag.
	// The map structure is: tagName -> list of entityIDs
	tagIndex map[string][]EntityID

	// Template index maps template names to a set of entity IDs that use that template.
	// The map structure is: templateName -> list of entityIDs
	templateIndex map[string][]EntityID

	// lazyTagTemplateIndex indicates whether the tag/template indexes are stale because
	// entities were created without eagerly updating lookup metadata.
	lazyTagTemplateIndex bool
}

func NewEntityIndex() entityIndex {
	return entityIndex{
		tagIndex:             make(map[string][]EntityID),
		templateIndex:        make(map[string][]EntityID),
		lazyTagTemplateIndex: false,
	}
}

func (idx *entityIndex) AddTag(entityID EntityID, tagName string) {
	idx.addToIndexString(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) RemoveTag(entityID EntityID, tagName string) {
	idx.removeFromIndexString(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) AddTemplate(entityID EntityID, templateName string) {
	idx.addToIndexString(idx.templateIndex, templateName, entityID)
}

func (idx *entityIndex) RemoveTemplate(entityID EntityID, templateName string) {
	idx.removeFromIndexString(idx.templateIndex, templateName, entityID)
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

func (idx *entityIndex) GetEntitiesWithTag(tagName string) []EntityID {
	return idx.getFromIndexString(idx.tagIndex, tagName)
}

func (idx *entityIndex) GetEntitiesWithTemplate(templateName string) []EntityID {
	return idx.getFromIndexString(idx.templateIndex, templateName)
}

func (idx *entityIndex) addToIndex(index map[reflect.Type][]EntityID, key reflect.Type, entityID EntityID) {
	if _, exists := index[key]; !exists {
		index[key] = make([]EntityID, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) addToIndexString(index map[string][]EntityID, key string, entityID EntityID) {
	if _, exists := index[key]; !exists {
		index[key] = make([]EntityID, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) rebuildTagAndTemplateIndex(entities map[EntityID]*entity) {
	idx.tagIndex = make(map[string][]EntityID)
	idx.templateIndex = make(map[string][]EntityID)

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

func (idx *entityIndex) removeFromIndex(index map[reflect.Type][]EntityID, key reflect.Type, entityID EntityID) {
	index[key] = removeEntityID(index[key], entityID)
}

func (idx *entityIndex) removeFromIndexString(index map[string][]EntityID, key string, entityID EntityID) {
	index[key] = removeEntityID(index[key], entityID)
}

func removeEntityID(entities []EntityID, entityID EntityID) []EntityID {
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

func (idx *entityIndex) getFromIndex(index map[reflect.Type][]EntityID, key reflect.Type) []EntityID {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

func (idx *entityIndex) getFromIndexString(index map[string][]EntityID, key string) []EntityID {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

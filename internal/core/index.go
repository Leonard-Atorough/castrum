package core

import (
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
	idx.addToIndex(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) RemoveTag(entityID EntityID, tagName string) {
	idx.removeFromIndex(idx.tagIndex, tagName, entityID)
}

func (idx *entityIndex) AddTemplate(entityID EntityID, templateName string) {
	idx.addToIndex(idx.templateIndex, templateName, entityID)
}

func (idx *entityIndex) RemoveTemplate(entityID EntityID, templateName string) {
	idx.removeFromIndex(idx.templateIndex, templateName, entityID)
}

func (idx *entityIndex) UpdateTag(entity *Entity, tagName string, add bool) {
	if add {
		idx.addToIndex(idx.tagIndex, tagName, entity.id)
	} else {
		idx.removeFromIndex(idx.tagIndex, tagName, entity.id)
	}
}

func (idx *entityIndex) UpdateTemplate(entity *Entity, oldTemplate string, newTemplate string) {
	if oldTemplate != "" {
		idx.removeFromIndex(idx.templateIndex, oldTemplate, entity.id)
	}
	if newTemplate != "" {
		idx.addToIndex(idx.templateIndex, newTemplate, entity.id)
	}
}

func (idx *entityIndex) GetEntitiesWithTag(tagName string) []EntityID {
	return idx.getFromIndex(idx.tagIndex, tagName)
}

func (idx *entityIndex) GetEntitiesWithTemplate(templateName string) []EntityID {
	return idx.getFromIndex(idx.templateIndex, templateName)
}

func (idx *entityIndex) addToIndex(index map[string][]EntityID, key string, entityID EntityID) {
	if _, exists := index[key]; !exists {
		index[key] = make([]EntityID, 0)
	}
	if !slices.Contains(index[key], entityID) {
		index[key] = append(index[key], entityID)
	}
}

func (idx *entityIndex) rebuildTagAndTemplateIndex(entities map[EntityID]*Entity) {
	idx.tagIndex = make(map[string][]EntityID)
	idx.templateIndex = make(map[string][]EntityID)

	for entityID, entity := range entities {
		if entity == nil {
			continue
		}
		if entity.template != "" {
			idx.addToIndex(idx.templateIndex, entity.template, entityID)
		}
		for tag := range entity.tags {
			idx.addToIndex(idx.tagIndex, tag, entityID)
		}
	}
	idx.lazyTagTemplateIndex = false
}

func (idx *entityIndex) removeFromIndex(index map[string][]EntityID, key string, entityID EntityID) {
	entities, exists := index[key]
	if !exists {
		return
	}
	for i, id := range entities {
		if id == entityID {
			entities = append(entities[:i], entities[i+1:]...)
			break
		}
	}
	if len(entities) == 0 {
		delete(index, key)
	} else {
		index[key] = entities
	}
}

func (idx *entityIndex) getFromIndex(index map[string][]EntityID, key string) []EntityID {
	value, ok := index[key]
	if !ok {
		return nil
	}
	return value
}

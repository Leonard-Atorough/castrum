package components

import "fmt"

// SceneTag marks which scene an entity belongs to, for query-time scene
// filtering via [ecs.Query.InScene]. Every entity in a scene gets one
// SceneTag; the SceneID identifies which scene.
type SceneTag struct {
	SceneID string
}

// NewSceneTag creates a SceneTag with the given scene ID.
func NewSceneTag(sceneID string) SceneTag {
	return SceneTag{SceneID: sceneID}
}

// Validate checks that the SceneID is non-empty.
func (s SceneTag) Validate() error {
	if s.SceneID == "" {
		return fmt.Errorf("sceneID must be non-empty")
	}
	return nil
}

// Serialize converts the SceneTag to a map suitable for blueprint
// serialization or save-game storage.
func (s SceneTag) Serialize() (map[string]any, error) {
	return map[string]any{"sceneID": s.SceneID}, nil
}

// Deserialize populates a SceneTag from a serialized map, returning the
// reconstructed component. The result is validated before returning.
func (s SceneTag) Deserialize(data map[string]any) (SceneTag, error) {
	if v, ok := data["sceneID"].(string); ok {
		s.SceneID = v
	}
	return s, s.Validate()
}

// Tag is a generic string marker for entities. Use it for ad-hoc grouping
// and querying — e.g., tagging entities as "enemy", "pickup", "platform".
// For scene membership, use [SceneTag] instead.
type Tag struct {
	Name string
}

// NewTag creates a Tag with the given name.
func NewTag(name string) Tag {
	return Tag{Name: name}
}

// Validate checks that the Tag name is non-empty.
func (t Tag) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tag name must be non-empty")
	}
	return nil
}

// Serialize converts the Tag to a map suitable for blueprint serialization
// or save-game storage.
func (t Tag) Serialize() (map[string]any, error) {
	return map[string]any{"name": t.Name}, nil
}

// Deserialize populates a Tag from a serialized map, returning the
// reconstructed component. The result is validated before returning.
func (t Tag) Deserialize(data map[string]any) (Tag, error) {
	if v, ok := data["name"].(string); ok {
		t.Name = v
	}
	return t, t.Validate()
}

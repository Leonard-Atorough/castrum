package components

import "testing"

func TestNewSceneTag(t *testing.T) {
	tag := NewSceneTag("level1")
	if tag.SceneID != "level1" {
		t.Errorf("SceneID = %q, want %q", tag.SceneID, "level1")
	}
}

func TestSceneTagValidate(t *testing.T) {
	if err := (SceneTag{SceneID: "a"}).Validate(); err != nil {
		t.Errorf("valid tag: unexpected error: %v", err)
	}
	if err := (SceneTag{SceneID: ""}).Validate(); err == nil {
		t.Error("empty SceneID: expected error, got nil")
	}
}

func TestSceneTagSerializeDeserializeRoundTrip(t *testing.T) {
	original := NewSceneTag("level1")
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	reconstructed, err := SceneTag{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}
	if reconstructed != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", reconstructed, original)
	}
}

func TestSceneTagDeserializeRejectsEmptyID(t *testing.T) {
	_, err := SceneTag{}.Deserialize(map[string]any{"sceneID": ""})
	if err == nil {
		t.Error("expected error for empty sceneID")
	}
}

func TestNewTag(t *testing.T) {
	tag := NewTag("enemy")
	if tag.Name != "enemy" {
		t.Errorf("Name = %q, want %q", tag.Name, "enemy")
	}
}

func TestTagValidate(t *testing.T) {
	if err := (Tag{Name: "a"}).Validate(); err != nil {
		t.Errorf("valid tag: unexpected error: %v", err)
	}
	if err := (Tag{Name: ""}).Validate(); err == nil {
		t.Error("empty name: expected error, got nil")
	}
}

func TestTagSerializeDeserializeRoundTrip(t *testing.T) {
	original := NewTag("pickup")
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	reconstructed, err := Tag{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}
	if reconstructed != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", reconstructed, original)
	}
}

func TestTagDeserializeRejectsEmptyName(t *testing.T) {
	_, err := Tag{}.Deserialize(map[string]any{"name": ""})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

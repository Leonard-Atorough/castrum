package components

import (
	"reflect"
	"testing"
)

func TestNewAnimation(t *testing.T) {
	tests := []struct {
		name     string
		clipID   string
		autoplay bool
	}{
		{"with autoplay", "clip1", true},
		{"without autoplay", "clip2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anim := NewAnimation(tt.clipID, tt.autoplay)
			if anim.ClipID != tt.clipID {
				t.Errorf("ClipID = %q, want %q", anim.ClipID, tt.clipID)
			}
			if anim.Playing != tt.autoplay {
				t.Errorf("Playing = %v, want %v", anim.Playing, tt.autoplay)
			}
			if anim.PlaybackSpeed != 1.0 {
				t.Errorf("PlaybackSpeed = %v, want 1.0", anim.PlaybackSpeed)
			}
			if anim.FrameIndex != 0 {
				t.Errorf("FrameIndex = %d, want 0", anim.FrameIndex)
			}
			if anim.FrameTime != 0 {
				t.Errorf("FrameTime = %v, want 0", anim.FrameTime)
			}
		})
	}
}

func TestAnimationValidate(t *testing.T) {
	tests := []struct {
		name    string
		anim    Animation
		wantErr bool
	}{
		{"valid", Animation{ClipID: "clip", FrameIndex: 0, FrameTime: 0, PlaybackSpeed: 1.0}, false},
		{"valid with runtime state", Animation{ClipID: "clip", FrameIndex: 5, FrameTime: 0.3, Playing: true, PlaybackSpeed: 2.0}, false},
		{"empty clipID", Animation{ClipID: "", PlaybackSpeed: 1.0}, true},
		{"negative frameIndex", Animation{ClipID: "clip", FrameIndex: -1, PlaybackSpeed: 1.0}, true},
		{"negative frameTime", Animation{ClipID: "clip", FrameTime: -0.1, PlaybackSpeed: 1.0}, true},
		{"zero playbackSpeed", Animation{ClipID: "clip", PlaybackSpeed: 0}, true},
		{"negative playbackSpeed", Animation{ClipID: "clip", PlaybackSpeed: -1.0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.anim.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnimationSerializeDeserializeRoundTrip(t *testing.T) {
	original := Animation{
		ClipID:        "walk",
		FrameIndex:    3,
		FrameTime:     0.15,
		Playing:       true,
		PlaybackSpeed: 1.5,
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := original.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed != original {
		t.Errorf("round-trip mismatch:\n  got  = %+v\n  want = %+v", reconstructed, original)
	}
}

func TestAnimationSerializeIncludesAllFields(t *testing.T) {
	anim := Animation{
		ClipID:        "idle",
		FrameIndex:    2,
		FrameTime:     0.05,
		Playing:       false,
		PlaybackSpeed: 0.5,
	}

	data, err := anim.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	expected := map[string]any{
		"clipID":        "idle",
		"frameIndex":    2,
		"frameTime":     0.05,
		"playing":       false,
		"playbackSpeed": 0.5,
	}

	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Serialize() = %+v, want %+v", data, expected)
	}
}

func TestAnimationDeserializeMissingFieldsUsesDefaults(t *testing.T) {
	// Only clipID and playbackSpeed are provided; other fields default to
	// zero values. playbackSpeed must be provided because Deserialize
	// validates and zero speed is invalid.
	data := map[string]any{
		"clipID":        "partial",
		"playbackSpeed": float64(1.0),
	}

	anim, err := Animation{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if anim.ClipID != "partial" {
		t.Errorf("ClipID = %q, want %q", anim.ClipID, "partial")
	}
	if anim.FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 (zero value)", anim.FrameIndex)
	}
	if anim.FrameTime != 0 {
		t.Errorf("FrameTime = %v, want 0 (zero value)", anim.FrameTime)
	}
	if anim.Playing {
		t.Errorf("Playing = true, want false (zero value)")
	}
}

func TestAnimationDeserializeRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{"empty clipID", map[string]any{"clipID": ""}},
		{"negative frameIndex", map[string]any{"clipID": "clip", "frameIndex": float64(-1)}},
		{"zero playbackSpeed", map[string]any{"clipID": "clip", "playbackSpeed": float64(0)}},
		{"missing clipID", map[string]any{"frameIndex": float64(1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Animation{}.Deserialize(tt.data)
			if err == nil {
				t.Error("Deserialize() expected error, got nil")
			}
		})
	}
}

func TestAnimationDeserializeHandlesFloat64FromJSON(t *testing.T) {
	// JSON unmarshals all numbers as float64. Ensure int fields convert correctly.
	data := map[string]any{
		"clipID":        "test",
		"frameIndex":    float64(42),
		"frameTime":     float64(0.25),
		"playing":       true,
		"playbackSpeed": float64(2.0),
	}

	anim, err := Animation{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if anim.FrameIndex != 42 {
		t.Errorf("FrameIndex = %d, want 42", anim.FrameIndex)
	}
	if anim.FrameTime != 0.25 {
		t.Errorf("FrameTime = %v, want 0.25", anim.FrameTime)
	}
	if anim.PlaybackSpeed != 2.0 {
		t.Errorf("PlaybackSpeed = %v, want 2.0", anim.PlaybackSpeed)
	}
}

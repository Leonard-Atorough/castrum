package components

import "testing"

func TestNewTimer(t *testing.T) {
	timer := NewTimer("test_timer", 12.5, false, false)
	if timer.ID != "test_timer" {
		t.Errorf("ID = %q, want %q", timer.ID, "test_timer")
	}
	if timer.Duration != 12.5 {
		t.Errorf("Duration = %v, want 12.5", timer.Duration)
	}
	if timer.Running {
		t.Errorf("Running = true, want false")
	}
	if timer.Once {
		t.Errorf("Once = true, want false")
	}
	if timer.ElapsedTime != 0 {
		t.Errorf("ElapsedTime = %v, want 0", timer.ElapsedTime)
	}
}

func TestNewTimerDefaultsNonPositiveDuration(t *testing.T) {
	timer := NewTimer("t", 0, true, false)
	if timer.Duration != 1.0 {
		t.Errorf("Duration = %v, want 1.0 (default)", timer.Duration)
	}
	timer = NewTimer("t", -5, true, false)
	if timer.Duration != 1.0 {
		t.Errorf("Duration = %v, want 1.0 (default)", timer.Duration)
	}
}

func TestTimerStartStopResume(t *testing.T) {
	timer := NewTimer("t", 5, false, false)

	timer.Start()
	if !timer.Running || timer.ElapsedTime != 0 {
		t.Errorf("after Start: Running=%v ElapsedTime=%v, want true/0", timer.Running, timer.ElapsedTime)
	}

	timer.ElapsedTime = 2.5
	timer.Stop()
	if timer.Running {
		t.Errorf("after Stop: Running=true, want false")
	}
	if timer.ElapsedTime != 2.5 {
		t.Errorf("after Stop: ElapsedTime=%v, want 2.5 (preserved)", timer.ElapsedTime)
	}

	timer.Resume()
	if !timer.Running {
		t.Errorf("after Resume: Running=false, want true")
	}
	if timer.ElapsedTime != 2.5 {
		t.Errorf("after Resume: ElapsedTime=%v, want 2.5 (preserved)", timer.ElapsedTime)
	}
}

func TestTimerValidate(t *testing.T) {
	tests := []struct {
		name    string
		timer   Timer
		wantErr bool
	}{
		{"valid", Timer{ID: "t", Duration: 1.0}, false},
		{"valid with state", Timer{ID: "t", Duration: 5.0, ElapsedTime: 2.0, Running: true, Once: true}, false},
		{"empty ID", Timer{ID: "", Duration: 1.0}, true},
		{"zero duration", Timer{ID: "t", Duration: 0}, true},
		{"negative duration", Timer{ID: "t", Duration: -1.0}, true},
		{"negative elapsedTime", Timer{ID: "t", Duration: 1.0, ElapsedTime: -0.1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.timer.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimerSerializeDeserializeRoundTrip(t *testing.T) {
	original := Timer{
		ID:          "spawn",
		Duration:    3.0,
		ElapsedTime: 1.5,
		Running:     true,
		Once:        false,
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	reconstructed, err := Timer{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if reconstructed != original {
		t.Errorf("round-trip mismatch:\n  got  = %+v\n  want = %+v", reconstructed, original)
	}
}

func TestTimerDeserializeRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{"empty ID", map[string]any{"id": "", "duration": float64(1.0)}},
		{"zero duration", map[string]any{"id": "t", "duration": float64(0)}},
		{"negative elapsedTime", map[string]any{"id": "t", "duration": float64(1.0), "elapsedTime": float64(-1)}},
		{"missing ID", map[string]any{"duration": float64(1.0)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Timer{}.Deserialize(tt.data)
			if err == nil {
				t.Error("Deserialize() expected error, got nil")
			}
		})
	}
}

func TestTimerDeserializeFloat64FromJSON(t *testing.T) {
	data := map[string]any{
		"id":          "test",
		"duration":    float64(5.0),
		"elapsedTime": float64(2.5),
		"running":     true,
		"once":        true,
	}

	timer, err := Timer{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if timer.Duration != 5.0 {
		t.Errorf("Duration = %v, want 5.0", timer.Duration)
	}
	if timer.ElapsedTime != 2.5 {
		t.Errorf("ElapsedTime = %v, want 2.5", timer.ElapsedTime)
	}
}

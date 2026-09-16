package components

import "fmt"

// TimerID uniquely identifies a timer within an entity. Multiple timers can
// coexist on the same entity as long as they have distinct IDs.
type TimerID string

// Timer tracks elapsed time and fires when its Duration is reached. A Timer
// can be one-shot (fires once, then the TimerSystem removes the component) or
// repeating (resets ElapsedTime and continues firing).
//
// Field roles:
//   - ID and Duration are configuration set at creation time.
//   - ElapsedTime and Running are runtime state mutated by the TimerSystem
//     via the get-mutate-set pattern.
//   - Once is configuration that controls whether the timer is removed after
//     firing.
//
// The TimerSystem accumulates ElapsedTime each tick for running timers. When
// ElapsedTime >= Duration, a TimerCompletedEvent is emitted. One-shot timers
// are then removed; repeating timers reset ElapsedTime to zero.
type Timer struct {
	ID          TimerID  // unique identifier for this timer
	Duration    float64  // how long the timer runs before firing (seconds)
	ElapsedTime float64  // accumulated time since the timer started (seconds)
	Running     bool     // whether the timer is currently accumulating time
	Once        bool     // true = fire once and remove; false = repeating
}

// NewTimer creates a Timer component with the given ID and duration. If
// duration is zero or negative, it defaults to 1 second. When running is
// true, the timer starts accumulating immediately.
func NewTimer(id string, duration float64, running, fireOnce bool) Timer {
	if duration <= 0 {
		duration = 1.0
	}
	return Timer{
		ID:          TimerID(id),
		Duration:    duration,
		ElapsedTime: 0,
		Running:     running,
		Once:        fireOnce,
	}
}

// Validate checks that the timer has a non-empty ID and a positive duration.
func (t Timer) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("timer ID must be non-empty")
	}
	if t.Duration <= 0 {
		return fmt.Errorf("timer duration must be positive, got %v", t.Duration)
	}
	if t.ElapsedTime < 0 {
		return fmt.Errorf("timer elapsedTime cannot be negative, got %v", t.ElapsedTime)
	}
	return nil
}

// Serialize converts the Timer to a map suitable for blueprint serialization
// or save-game storage. All fields are included.
func (t Timer) Serialize() (map[string]any, error) {
	return map[string]any{
		"id":          string(t.ID),
		"duration":    t.Duration,
		"elapsedTime": t.ElapsedTime,
		"running":     t.Running,
		"once":        t.Once,
	}, nil
}

// Deserialize populates a Timer from a serialized map, returning the
// reconstructed component. The result is validated before returning.
func (t Timer) Deserialize(data map[string]any) (Timer, error) {
	if v, ok := data["id"].(string); ok {
		t.ID = TimerID(v)
	}
	if v, ok := data["duration"].(float64); ok {
		t.Duration = v
	}
	if v, ok := data["elapsedTime"].(float64); ok {
		t.ElapsedTime = v
	}
	if v, ok := data["running"].(bool); ok {
		t.Running = v
	}
	if v, ok := data["once"].(bool); ok {
		t.Once = v
	}
	return t, t.Validate()
}

// Start begins the timer, resetting elapsed time to zero. Uses a pointer
// receiver because it mutates the timer.
func (t *Timer) Start() {
	t.Running = true
	t.ElapsedTime = 0
}

// Stop pauses the timer without resetting elapsed time. Resume can be called
// to continue from the same elapsed time. Uses a pointer receiver because it
// mutates the timer.
func (t *Timer) Stop() {
	t.Running = false
}

// Resume continues a stopped timer from its current elapsed time. Uses a
// pointer receiver because it mutates the timer.
func (t *Timer) Resume() {
	t.Running = true
}

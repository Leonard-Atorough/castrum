// Package timers provides a timer system for managing one-shot and repeating timers
// within a game loop. Timers accumulate delta time from frame updates and fire callbacks
// when their duration is reached.
//
// The Manager coordinates multiple timers and should be called once per frame via Update.
// Timers are thread-safe and can be safely accessed from multiple goroutines.
//
// Example usage:
//
//	manager := NewManager()
//	timer := manager.CreateTimer(2.0, true, true, func() {
//		fmt.Println("Timer fired!")
//	})
//	// In game loop:
//	manager.Update(deltaTime)
package timers

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// TimerID uniquely identifies a timer within a Manager.
type TimerID uint64

// Timer tracks elapsed time and fires a callback when its duration expires.
// A timer can be one-shot (fires once and stops) or repeating (resets and fires again).
// Timers are thread-safe and can be safely started, stopped, and queried from multiple goroutines.
//
// Timers accumulate delta time passed to Update. When accumulated time reaches the duration,
// the timer fires and either stops (one-shot) or resets for the next interval (repeating).
type Timer struct {
	// Unique identifier for the timer
	id TimerID
	// Duration for which the timer runs
	duration float64
	// Elapsed time since the timer started
	elapsed float64
	// Indicates whether the timer is currently running
	running bool
	// Indicates whether the timer has been cancelled
	cancelled bool
	// Indicates whether the timer should start automatically upon creation
	autoStart bool
	// Indicates whether the timer is a fire once timer or a repeating timer
	once bool
	// optional callback function to be called when the timer completes
	timerFunc func()
	// optional callback function to be called when the timer is cancelled
	cancelFunc func()
	// protects Timer state during concurrent access
	mu sync.Mutex
}

// NewTimer creates a new timer with the given parameters.
// If autoStart is true, the timer begins running immediately.
// The timerFunc callback (if not nil) will be invoked by the Manager after calling Update.
func NewTimer(id TimerID, duration float64, once bool, autoStart bool, timerFunc func()) *Timer {
	return &Timer{
		id:        id,
		duration:  duration,
		elapsed:   0,
		running:   autoStart,
		autoStart: autoStart,
		once:      once,
		timerFunc: timerFunc,
	}
}

// Start begins the timer, resetting elapsed time to zero.
func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = true
	t.elapsed = 0
}

// Stop pauses the timer without resetting elapsed time.
// Resume can be called to continue from the same elapsed time.
func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
}

// Resume continues a stopped timer from its current elapsed time.
func (t *Timer) Resume() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = true
}

// Cancel stops the timer and resets its state. The cancel callback (if set)
// is invoked immediately. After cancellation, the timer will not fire.
func (t *Timer) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	t.elapsed = 0
	t.cancelled = true
	// Call cancel callback if set
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}

// Update advances the timer by the given delta time in seconds.
// It returns (completed, shouldFire) where completed indicates the timer reached
// its duration and shouldFire indicates the callback should be invoked.
// The Manager calls this and handles callback execution and timer cleanup.
func (t *Timer) Update(deltaTime float64) (completed bool, shouldFire bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running || t.cancelled {
		return false, false
	}

	t.elapsed += deltaTime

	if t.elapsed < t.duration {
		return false, false
	}

	shouldFire = t.timerFunc != nil

	if t.once {
		t.cancelled = true
		t.running = false
		t.elapsed = 0
	}

	if !t.once {
		t.elapsed = 0 // Reset elapsed time for repeating timer
	}

	return true, shouldFire
}

// AutoStart returns whether this timer auto-starts when created.
func (t *Timer) AutoStart() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.autoStart
}

// SetAutoStart sets whether the timer should auto-start when created.
func (t *Timer) SetAutoStart(autoStart bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.autoStart = autoStart
}

// SetCancelFunc registers an optional callback to be invoked when the timer is cancelled.
// This is useful for cleanup tasks (e.g., releasing resources).
func (t *Timer) SetCancelFunc(cancelFunc func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelFunc = cancelFunc
}

// IsCancelled returns true if the timer has been cancelled.
func (t *Timer) IsCancelled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cancelled
}

// IsRunning returns true if the timer is currently running.
func (t *Timer) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// IsOnce returns true if this is a one-shot timer (fires once then stops).
func (t *Timer) IsOnce() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.once
}

// ID returns the unique identifier of this timer.
func (t *Timer) ID() TimerID {
	return t.id // ID is immutable after creation, no lock needed
}

// Duration returns the duration this timer will run for.
func (t *Timer) Duration() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.duration
}

// Elapsed returns the accumulated elapsed time on this timer.
func (t *Timer) Elapsed() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.elapsed
}

// Manager coordinates multiple timers and manages their lifecycle.
// The Manager should be called once per frame via Update to advance all timers
// and invoke their callbacks. Callbacks are executed with panic recovery to prevent
// one failing callback from stopping other timers.
//
// Manager is thread-safe for creating, removing, and querying timers.
type Manager struct {
	// Map to store timers with their unique IDs
	timers map[TimerID]*Timer
	// Counter to generate unique TimerIDs
	nextID TimerID
	// Indicates whether the manager is paused (all timers suspended)
	paused bool
	// protects manager state
	mu sync.Mutex
	// Pre-allocated slices for reuse (to reduce GC pressure)
	callbacks    []func()
	cleanupIDs   []TimerID
	maxTimerHint int
}

// NewManager creates a new timer manager.
func NewManager() *Manager {
	return &Manager{
		timers:       make(map[TimerID]*Timer),
		nextID:       1,
		maxTimerHint: 64, // Initial hint for pre-allocation
		callbacks:    make([]func(), 0, 64),
		cleanupIDs:   make([]TimerID, 0, 64),
	}
}

// CreateTimer creates and registers a new timer with the manager.
// The timer is immediately available for use and will be updated when Manager.Update is called.
func (tm *Manager) CreateTimer(duration float64, once bool, autoStart bool, callbackFunc func()) *Timer {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timerID := tm.nextID
	tm.nextID++
	timer := NewTimer(timerID, duration, once, autoStart, callbackFunc)
	tm.timers[timerID] = timer
	return timer
}

// Update advances all timers by the given delta time in seconds and invokes their callbacks.
// Callbacks are executed with panic recovery; a panicking callback will not prevent other
// callbacks from executing. Expired one-shot timers and cancelled timers are automatically cleaned up.
// If deltaTime is negative, Update returns early without processing.
//
// Update should be called once per frame from the main game loop.
// While paused via Pause(), Update skips all timer processing.
func (tm *Manager) Update(deltaTime float64) {
	if deltaTime < 0 {
		return
	}

	tm.mu.Lock()
	if tm.paused {
		tm.mu.Unlock()
		return
	}

	// Reset pre-allocated slices
	tm.callbacks = tm.callbacks[:0]
	tm.cleanupIDs = tm.cleanupIDs[:0]

	for _, timer := range tm.timers {
		completed, shouldFire := timer.Update(deltaTime)
		if !completed {
			continue
		}
		if shouldFire {
			tm.callbacks = append(tm.callbacks, timer.timerFunc)
		}
		if timer.IsOnce() || timer.IsCancelled() {
			tm.cleanupIDs = append(tm.cleanupIDs, timer.ID())
		}
	}

	// Copy slices before unlocking
	callbacksCopy := make([]func(), len(tm.callbacks))
	copy(callbacksCopy, tm.callbacks)
	cleanupIDsCopy := make([]TimerID, len(tm.cleanupIDs))
	copy(cleanupIDsCopy, tm.cleanupIDs)

	tm.mu.Unlock()

	// Execute callbacks with panic recovery
	for _, cb := range callbacksCopy {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("timer callback panic: %v", r)
				}
			}()
			cb()
		}()
	}

	// Clean up expired/cancelled timers
	for _, timerID := range cleanupIDsCopy {
		tm.RemoveTimer(timerID)
	}
}

// RemoveTimer cancels and removes the timer from the manager.
// If the timer has a cancel callback set, it is invoked.
// Removing a non-existent timer returns an error.
func (tm *Manager) RemoveTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, err := tm.lookupTimer(timerID)
	if err != nil {
		return err
	}
	timer.Cancel()

	delete(tm.timers, timerID)
	return nil
}

// HasTimer returns true if a timer with the given ID exists in the manager.
func (tm *Manager) HasTimer(timerID TimerID) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, exists := tm.timers[timerID]
	return exists
}

// Pause suspends all timers in the manager. While paused, Update will skip processing.
// Timers maintain their current elapsed time and can be resumed with Resume.
func (tm *Manager) Pause() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.paused = true
}

// Resume resumes the manager after being paused.
func (tm *Manager) Resume() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.paused = false
}

// IsPaused returns true if the manager is currently paused.
func (tm *Manager) IsPaused() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.paused
}

func (tm *Manager) lookupTimer(timerID TimerID) (*Timer, error) {
	timer, exists := tm.timers[timerID]
	if !exists {
		return nil, &TimerError{TimerID: timerID, Op: "lookup", Err: ErrTimerNotFound}
	}
	return timer, nil
}

// Errors returned by timer operations.
var (
	ErrTimerNotFound        = errors.New("timer not found")
	ErrTimerInvalidDuration = errors.New("invalid timer duration")
	ErrTimerAlreadyRunning  = errors.New("timer already running")
	ErrTimerAlreadyStopped  = errors.New("timer already stopped")
)

// TimerError describes an error that occurred during a timer operation.
// It includes the timer ID, operation name, and underlying error.
type TimerError struct {
	TimerID TimerID
	Op      string
	Err     error
}

// Error returns the string representation of the timer error.
func (e *TimerError) Error() string {
	return fmt.Sprintf("timer %d: %s: %v", e.TimerID, e.Op, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *TimerError) Unwrap() error {
	return e.Err
}

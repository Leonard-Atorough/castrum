package timers

import (
	"sync"
)

type Manager struct {
	// Map to store timers with their unique IDs
	timers map[TimerID]*Timer
	// Counter to generate unique TimerIDs
	nextID TimerID
	mu     sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		timers: make(map[TimerID]*Timer),
		nextID: 1,
	}
}

func (tm *Manager) CreateTimer(duration float64, once bool, autoStart bool, callbackFunc func()) *Timer {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timerID := tm.nextID
	tm.nextID++
	timer := NewTimer(timerID, duration, once, autoStart, callbackFunc)
	tm.timers[timerID] = timer
	return timer
}

func (tm *Manager) Update(deltaTime float64) {
	if deltaTime < 0 {
		return
	}

	tm.mu.Lock()

	var callbacks []func()
	var cleanupTimers []TimerID

	for _, timer := range tm.timers {
		completed, shouldFire := timer.Update(deltaTime)
		if !completed {
			continue
		}
		if shouldFire {
			callbacks = append(callbacks, timer.timerFunc)
		}
		if timer.IsOnce() || timer.cancelled {
			cleanupTimers = append(cleanupTimers, timer.ID())
		}
	}

	tm.mu.Unlock()

	for _, cb := range callbacks {
		cb()
	}

	for _, timerID := range cleanupTimers {
		tm.RemoveTimer(timerID)
	}
}

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

func (tm *Manager) HasTimer(timerID TimerID) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, exists := tm.timers[timerID]
	return exists
}

func (tm *Manager) lookupTimer(timerID TimerID) (*Timer, error) {
	timer, exists := tm.timers[timerID]
	if !exists {
		return nil, &TimerError{TimerID: timerID, Op: "lookup", Err: ErrTimerNotFound}
	}
	return timer, nil
}

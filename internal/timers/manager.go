package timers

import (
	"sync"
)

type TimerManager struct {
	// Map to store timers with their unique IDs
	timers map[TimerID]*Timer
	// Counter to generate unique TimerIDs
	nextID TimerID
	mu     sync.Mutex
}

func NewTimerManager() *TimerManager {
	return &TimerManager{
		timers: make(map[TimerID]*Timer),
		nextID: 1,
	}
}

func (tm *TimerManager) CreateTimer(duration float64, once bool, autoStart bool, callbackFunc func()) TimerID {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timerID := tm.nextID
	tm.nextID++
	timer := NewTimer(timerID, duration, once, autoStart, callbackFunc)
	tm.timers[timerID] = timer
	return timerID
}

func (tm *TimerManager) StartTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, err := tm.lookupTimer(timerID)
	if err != nil {
		return err
	}
	if timer.IsRunning() {
		return &TimerError{TimerID: timerID, Op: "start", Err: ErrTimerAlreadyRunning}
	}
	timer.Start()
	return nil
}

func (tm *TimerManager) ResumeTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, err := tm.lookupTimer(timerID)
	if err != nil {
		return err
	}
	if timer.IsRunning() {
		return &TimerError{TimerID: timerID, Op: "resume", Err: ErrTimerAlreadyRunning}
	}
	timer.Resume()
	return nil
}

func (tm *TimerManager) StopTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, err := tm.lookupTimer(timerID)
	if err != nil {
		return err
	}
	if !timer.IsRunning() {
		return &TimerError{TimerID: timerID, Op: "stop", Err: ErrTimerAlreadyStopped}
	}
	timer.Stop()
	return nil
}

func (tm *TimerManager) Update(deltaTime float64) {
	if deltaTime < 0 {
		return
	}

	tm.mu.Lock()

	var callbacks []func()
	var cleanupTimers []TimerID

	for _, timer := range tm.timers {
		if !timer.Update(deltaTime) {
			continue
		}
		if timer.timerFunc != nil {
			callbacks = append(callbacks, timer.timerFunc)
		}
		if timer.IsOnce() {
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

func (tm *TimerManager) RemoveTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, err := tm.lookupTimer(timerID)
	if err != nil {
		return err
	}
	timer.Stop()

	delete(tm.timers, timerID)
	return nil
}

func (tm *TimerManager) lookupTimer(timerID TimerID) (*Timer, error) {
	timer, exists := tm.timers[timerID]
	if !exists {
		return nil, &TimerError{TimerID: timerID, Op: "lookup", Err: ErrTimerNotFound}
	}
	return timer, nil
}

package timers

import (
	"fmt"
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

	timer, exists := tm.timers[timerID]
	if !exists {
		return fmt.Errorf("timer with ID %d does not exist", timerID)
	}
	timer.Start()
	return nil
}

func (tm *TimerManager) ResumeTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, exists := tm.timers[timerID]
	if !exists {
		return fmt.Errorf("timer with ID %d does not exist", timerID)
	}
	timer.Resume()
	return nil
}

func (tm *TimerManager) StopTimer(timerID TimerID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, exists := tm.timers[timerID]
	if !exists {
		return fmt.Errorf("timer with ID %d does not exist", timerID)
	}
	timer.Stop()
	return nil
}

func (tm *TimerManager) UpdateTimers(deltaTime float64) {
	tm.mu.Lock()

	var callbacks []func()
	var cleanupTimers []TimerID

	for _, timer := range tm.timers {
		if !timer.Update(deltaTime) {
			continue // Timer has not expired yet
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

	timer, exists := tm.timers[timerID]
	if !exists {
		return fmt.Errorf("timer with ID %d does not exist", timerID)
	}
	timer.Stop()

	delete(tm.timers, timerID)
	return nil
}

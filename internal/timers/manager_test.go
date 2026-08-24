package timers

import (
	"testing"
	"time"
)

func TestTimerManager_CreateStartStopAndRemove(t *testing.T) {
	manager := NewTimerManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timerID := manager.CreateTimer(0.25, true, false, callback)
	if timerID == 0 {
		t.Fatal("expected timer id to be greater than zero")
	}
	if _, exists := manager.timers[timerID]; !exists {
		t.Fatal("expected timer to be registered")
	}

	if err := manager.StartTimer(timerID); err != nil {
		t.Fatalf("start timer failed: %v", err)
	}
	if !manager.timers[timerID].IsRunning() {
		t.Fatal("timer should be running after StartTimer")
	}

	if err := manager.StopTimer(timerID); err != nil {
		t.Fatalf("stop timer failed: %v", err)
	}
	if manager.timers[timerID].IsRunning() {
		t.Fatal("timer should not be running after StopTimer")
	}

	if err := manager.RemoveTimer(timerID); err != nil {
		t.Fatalf("remove timer failed: %v", err)
	}
	if _, exists := manager.timers[timerID]; exists {
		t.Fatal("timer should be removed from manager")
	}
}

func TestTimerManager_UpdateTimersTriggersCallback(t *testing.T) {
	manager := NewTimerManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timerID := manager.CreateTimer(0.1, true, true, callback)
	manager.UpdateTimers(0.1)

	if callbackCalled != 1 {
		t.Fatalf("expected callback once, got %d", callbackCalled)
	}
	if _, exists := manager.timers[timerID]; exists {
		t.Fatal("one-shot timer should be cleaned up after triggering")
	}
}

func TestTimerManager_RepeatingTimerKeepsFiring(t *testing.T) {
	manager := NewTimerManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timerID := manager.CreateTimer(0.05, false, true, callback)
	if timerID == 0 {
		t.Fatal("timer id should be valid")
	}

	manager.UpdateTimers(0.05)
	if callbackCalled != 1 {
		t.Fatalf("expected first callback, got %d", callbackCalled)
	}
	manager.UpdateTimers(0.05)
	if callbackCalled != 2 {
		t.Fatalf("expected repeating callback twice, got %d", callbackCalled)
	}

	if err := manager.RemoveTimer(timerID); err != nil {
		t.Fatalf("remove timer failed: %v", err)
	}
}

func TestTimerManager_OneShotTimerAutoCleanupAfterExpiry(t *testing.T) {
	t.Run("with callback", func(t *testing.T) {
		manager := NewTimerManager()
		timerID := manager.CreateTimer(0.05, true, true, func() {})

		manager.UpdateTimers(0.05)

		if _, exists := manager.timers[timerID]; exists {
			t.Fatal("one-shot timer with callback should be removed automatically after expiry")
		}
	})

	t.Run("without callback", func(t *testing.T) {
		manager := NewTimerManager()
		timerID := manager.CreateTimer(0.05, true, true, nil)

		manager.UpdateTimers(0.05)

		if _, exists := manager.timers[timerID]; exists {
			t.Fatal("one-shot timer without callback should also be removed automatically after expiry")
		}
	})
}

func TestTimerManager_UnknownTimerIDErrors(t *testing.T) {
	manager := NewTimerManager()
	missingID := TimerID(999)

	if err := manager.StartTimer(missingID); err == nil {
		t.Fatal("expected StartTimer to fail for missing id")
	}
	if err := manager.StopTimer(missingID); err == nil {
		t.Fatal("expected StopTimer to fail for missing id")
	}
	if err := manager.ResumeTimer(missingID); err == nil {
		t.Fatal("expected ResumeTimer to fail for missing id")
	}
	if err := manager.RemoveTimer(missingID); err == nil {
		t.Fatal("expected RemoveTimer to fail for missing id")
	}
}

func TestTimerManager_CallbackCanMutateManagerWithoutDeadlock(t *testing.T) {
	manager := NewTimerManager()
	var callbackCalled int

	var timerID TimerID
	var callback func()
	callback = func() {
		callbackCalled++
		if err := manager.StopTimer(timerID); err != nil {
			t.Errorf("callback should be able to stop timer without deadlock: %v", err)
		}
	}

	timerID = manager.CreateTimer(0.05, true, true, callback)
	done := make(chan struct{})
	go func() {
		manager.UpdateTimers(0.05)
		close(done)
	}()

	select {
	case <-done:
		if callbackCalled != 1 {
			t.Fatalf("expected callback to run once, got %d", callbackCalled)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("UpdateTimers deadlocked while callback tried to mutate the manager")
	}
}

package timers

import (
	"errors"
	"testing"
	"time"
)

func TestTimerManager_CreateStartStopAndRemove(t *testing.T) {
	manager := NewManager()
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
	manager := NewManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timerID := manager.CreateTimer(0.1, true, true, callback)
	manager.Update(0.1)

	if callbackCalled != 1 {
		t.Fatalf("expected callback once, got %d", callbackCalled)
	}
	if _, exists := manager.timers[timerID]; exists {
		t.Fatal("one-shot timer should be cleaned up after triggering")
	}
}

func TestTimerManager_RepeatingTimerKeepsFiring(t *testing.T) {
	manager := NewManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timerID := manager.CreateTimer(0.05, false, true, callback)
	if timerID == 0 {
		t.Fatal("timer id should be valid")
	}

	manager.Update(0.05)
	if callbackCalled != 1 {
		t.Fatalf("expected first callback, got %d", callbackCalled)
	}
	manager.Update(0.05)
	if callbackCalled != 2 {
		t.Fatalf("expected repeating callback twice, got %d", callbackCalled)
	}

	if err := manager.RemoveTimer(timerID); err != nil {
		t.Fatalf("remove timer failed: %v", err)
	}
}

func TestTimerManager_OneShotTimerAutoCleanupAfterExpiry(t *testing.T) {
	t.Run("with callback", func(t *testing.T) {
		manager := NewManager()
		timerID := manager.CreateTimer(0.05, true, true, func() {})

		manager.Update(0.05)

		if _, exists := manager.timers[timerID]; exists {
			t.Fatal("one-shot timer with callback should be removed automatically after expiry")
		}
	})

	t.Run("without callback", func(t *testing.T) {
		manager := NewManager()
		timerID := manager.CreateTimer(0.05, true, true, nil)

		manager.Update(0.05)

		if _, exists := manager.timers[timerID]; exists {
			t.Fatal("one-shot timer without callback should also be removed automatically after expiry")
		}
	})
}

func TestTimerManager_UnknownTimerIDErrors(t *testing.T) {
	manager := NewManager()
	missingID := TimerID(999)

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{name: "start", call: func() error { return manager.StartTimer(missingID) }},
		{name: "stop", call: func() error { return manager.StopTimer(missingID) }},
		{name: "resume", call: func() error { return manager.ResumeTimer(missingID) }},
		{name: "remove", call: func() error { return manager.RemoveTimer(missingID) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrTimerNotFound) {
				t.Fatalf("expected ErrTimerNotFound, got %v", err)
			}
			var timerErr *TimerError
			if !errors.As(err, &timerErr) {
				t.Fatalf("expected TimerError wrapper, got %T: %v", err, err)
			}
			if timerErr.TimerID != missingID {
				t.Fatalf("expected wrapped timer id %d, got %d", missingID, timerErr.TimerID)
			}
		})
	}
}

func TestTimerManager_StateTransitionErrors(t *testing.T) {
	manager := NewManager()
	timerID := manager.CreateTimer(0.2, true, true, nil)

	if err := manager.StartTimer(timerID); err == nil || !errors.Is(err, ErrTimerAlreadyRunning) {
		t.Fatal("expected ErrTimerAlreadyRunning from StartTimer on an active timer")
	}

	if err := manager.StopTimer(timerID); err != nil {
		t.Fatalf("StopTimer should succeed on active timer: %v", err)
	}
	if err := manager.StopTimer(timerID); err == nil || !errors.Is(err, ErrTimerAlreadyStopped) {
		t.Fatal("expected ErrTimerAlreadyStopped from StopTimer on an inactive timer")
	}
}

func TestTimerManager_CallbackCanMutateManagerWithoutDeadlock(t *testing.T) {
	manager := NewManager()
	var callbackCalled int

	var timerID TimerID
	var callback func()
	callback = func() {
		callbackCalled++
		if err := manager.StartTimer(timerID); err != nil && !errors.Is(err, ErrTimerAlreadyRunning) {
			t.Fatalf("callback should be able to re-enter manager without deadlock: %v", err)
		}
	}

	timerID = manager.CreateTimer(0.05, true, true, callback)
	done := make(chan struct{})
	go func() {
		manager.Update(0.05)
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

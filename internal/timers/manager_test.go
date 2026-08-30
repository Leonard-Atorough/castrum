package timers

import (
	"testing"
)

func TestTimerManager_CreateAndRemove(t *testing.T) {
	manager := NewManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timer := manager.CreateTimer(0.25, true, false, callback)
	if timer == nil {
		t.Fatal("expected timer to be non-nil")
	}
	if _, exists := manager.timers[timer.ID()]; !exists {
		t.Fatal("expected timer to be registered")
	}

	if err := manager.RemoveTimer(timer.ID()); err != nil {
		t.Fatalf("remove timer failed: %v", err)
	}
	if _, exists := manager.timers[timer.ID()]; exists {
		t.Fatal("timer should be removed from manager")
	}
}

func TestTimerManager_UpdateTimersTriggersCallback(t *testing.T) {
	manager := NewManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timer := manager.CreateTimer(0.1, true, true, callback)
	manager.Update(0.1)

	if callbackCalled != 1 {
		t.Fatalf("expected callback once, got %d", callbackCalled)
	}
	if _, exists := manager.timers[timer.ID()]; exists {
		t.Fatal("one-shot timer should be cleaned up after triggering")
	}
}

func TestTimerManager_RepeatingTimerKeepsFiring(t *testing.T) {
	manager := NewManager()
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timer := manager.CreateTimer(0.05, false, true, callback)
	if timer == nil {
		t.Fatal("timer should be non-nil")
	}

	manager.Update(0.05)
	if callbackCalled != 1 {
		t.Fatalf("expected first callback, got %d", callbackCalled)
	}
	manager.Update(0.05)
	if callbackCalled != 2 {
		t.Fatalf("expected repeating callback twice, got %d", callbackCalled)
	}

	if err := manager.RemoveTimer(timer.ID()); err != nil {
		t.Fatalf("remove timer failed: %v", err)
	}
}

func TestTimerManager_OneShotTimerAutoCleanupAfterExpiry(t *testing.T) {
	t.Run("with callback", func(t *testing.T) {
		manager := NewManager()
		timer := manager.CreateTimer(0.05, true, true, func() {})

		manager.Update(0.05)

		if _, exists := manager.timers[timer.ID()]; exists {
			t.Fatal("one-shot timer with callback should be removed automatically after expiry")
		}
	})

	t.Run("without callback", func(t *testing.T) {
		manager := NewManager()
		timer := manager.CreateTimer(0.05, true, true, nil)

		manager.Update(0.05)

		if _, exists := manager.timers[timer.ID()]; exists {
			t.Fatal("one-shot timer without callback should also be removed automatically after expiry")
		}
	})
}

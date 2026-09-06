package timers

import (
	"sync"
	"testing"
)

func TestTimer_BasicLifecycle(t *testing.T) {
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timer := NewTimer(1, 0.5, true, false, callback)
	if timer == nil {
		t.Fatal("expected timer instance")
	}
	if timer.ID() != 1 {
		t.Fatalf("expected id 1, got %d", timer.ID())
	}
	if timer.Duration() != 0.5 {
		t.Fatalf("expected duration 0.5, got %f", timer.Duration())
	}
	if timer.IsRunning() {
		t.Fatal("timer should not auto-start when autoStart is false")
	}
	if timer.AutoStart() {
		t.Fatal("expected autoStart to be false")
	}

	timer.Start()
	if !timer.IsRunning() {
		t.Fatal("timer should be running after Start")
	}
	if timer.Elapsed() != 0 {
		t.Fatalf("elapsed should reset to zero on Start, got %f", timer.Elapsed())
	}

	timer.Stop()
	if timer.IsRunning() {
		t.Fatal("timer should not be running after Stop")
	}

	timer.Update(0.3)
	if callbackCalled != 0 {
		t.Fatalf("callback should not fire while timer is stopped, got %d calls", callbackCalled)
	}
}

func TestTimer_UpdateFiresWhenExpired(t *testing.T) {
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timer := NewTimer(2, 0.25, true, true, callback)
	if !timer.IsRunning() {
		t.Fatal("timer should auto-start")
	}

	if _, shouldFire := timer.Update(0.1); shouldFire {
		t.Fatal("timer should not fire before duration completes")
	}
	if callbackCalled != 0 {
		t.Fatalf("callback should not fire before expiration, got %d calls", callbackCalled)
	}

	if _, shouldFire := timer.Update(0.2); !shouldFire {
		t.Fatal("timer should fire once duration is reached")
	}
	if timer.IsRunning() {
		t.Fatal("one-shot timer should stop after triggering")
	}
	if callbackCalled != 0 {
		t.Fatalf("timer.Update should not invoke callback directly; got %d calls", callbackCalled)
	}
}

func TestTimer_UpdateRepeatingTimerResetsAndContinues(t *testing.T) {
	callbackCalled := 0
	callback := func() { callbackCalled++ }

	timer := NewTimer(3, 0.1, false, true, callback)
	if !timer.IsRunning() {
		t.Fatal("repeating timer should auto-start")
	}

	if _, shouldFire := timer.Update(0.1); !shouldFire {
		t.Fatal("timer should fire at its first duration boundary")
	}
	if callbackCalled != 0 {
		t.Fatalf("timer.Update should not call callback directly; got %d", callbackCalled)
	}
	if !timer.IsRunning() {
		t.Fatal("repeating timer should continue running after firing")
	}
	if timer.Elapsed() != 0 {
		t.Fatalf("repeating timer should reset elapsed after a cycle, got %f", timer.Elapsed())
	}

	if _, shouldFire := timer.Update(0.1); !shouldFire {
		t.Fatal("timer should fire again on the next interval")
	}
	if callbackCalled != 0 {
		t.Fatalf("timer.Update should not call callback directly on subsequent cycles; got %d", callbackCalled)
	}
}

func TestTimer_UnhappyPath_UnknownStateTransitions(t *testing.T) {
	timer := NewTimer(4, 1, false, false, nil)
	if timer.IsRunning() {
		t.Fatal("timer should start stopped")
	}

	if _, shouldFire := timer.Update(0.5); shouldFire {
		t.Fatal("stopped timer should not fire")
	}
	if timer.Elapsed() != 0 {
		t.Fatalf("stopped timer should not accumulate elapsed while inactive, got %f", timer.Elapsed())
	}
}

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

// Test cancel callback execution
func TestTimer_CancelCallbackFires(t *testing.T) {
	cancelCalled := 0
	cancelCallback := func() { cancelCalled++ }

	timer := NewTimer(1, 1.0, true, true, func() {})
	timer.SetCancelFunc(cancelCallback)

	if cancelCalled != 0 {
		t.Fatalf("cancel callback should not fire yet, got %d calls", cancelCalled)
	}

	timer.Cancel()
	if cancelCalled != 1 {
		t.Fatalf("cancel callback should have been called once, got %d calls", cancelCalled)
	}
}

func TestTimer_CancelCallbackOptional(t *testing.T) {
	timer := NewTimer(1, 1.0, true, true, func() {})
	// No cancel callback set - should not panic
	timer.Cancel()
	if !timer.IsCancelled() {
		t.Fatal("timer should be cancelled")
	}
}

func TestTimerManager_CallbackPanicRecovery(t *testing.T) {
	manager := NewManager()
	callbackCount := 0

	// Timer with panicking callback
	manager.CreateTimer(0.1, true, true, func() {
		panic("test panic")
	})

	// Timer with normal callback (should still fire)
	manager.CreateTimer(0.1, true, true, func() {
		callbackCount++
	})

	// Should not panic; both timers should be updated
	manager.Update(0.1)

	if callbackCount != 1 {
		t.Fatalf("normal callback should have fired despite panic in another, got %d", callbackCount)
	}
}

func TestTimerManager_PauseAndResume(t *testing.T) {
	manager := NewManager()
	callbackCount := 0
	callback := func() { callbackCount++ }

	manager.CreateTimer(0.1, false, true, callback)

	// First update should fire
	manager.Update(0.1)
	if callbackCount != 1 {
		t.Fatalf("expected callback once, got %d", callbackCount)
	}

	// Pause the manager
	manager.Pause()
	if !manager.IsPaused() {
		t.Fatal("manager should be paused")
	}

	// Update should be skipped while paused
	manager.Update(0.1)
	if callbackCount != 1 {
		t.Fatalf("callback should not fire while paused, got %d", callbackCount)
	}

	// Resume and verify callback fires again
	manager.Resume()
	if manager.IsPaused() {
		t.Fatal("manager should not be paused")
	}

	manager.Update(0.1)
	if callbackCount != 2 {
		t.Fatalf("expected callback twice after resume, got %d", callbackCount)
	}
}

func TestTimer_ThreadSafety_ConcurrentAccess(t *testing.T) {
	timer := NewTimer(1, 1.0, true, false, func() {})
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Spawn multiple goroutines accessing the same timer concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Try various operations concurrently
			if id%3 == 0 {
				timer.Start()
			} else if id%3 == 1 {
				timer.Stop()
			} else {
				timer.Update(0.01)
			}
		}(i)
	}

	// Also do read operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = timer.IsRunning()
			_ = timer.Elapsed()
			_ = timer.IsCancelled()
		}()
	}

	wg.Wait()
	close(errors)

	// If we get here without a race condition or panic, test passed
	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestTimerManager_ThreadSafety_ConcurrentCreation(t *testing.T) {
	manager := NewManager()
	var wg sync.WaitGroup
	timerCount := 0
	var mu sync.Mutex

	// Create timers concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.CreateTimer(0.1, true, false, func() {})
			mu.Lock()
			timerCount++
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(manager.timers) != timerCount {
		t.Fatalf("expected %d timers created, got %d", timerCount, len(manager.timers))
	}
}

func TestTimerManager_SlicePreallocation(t *testing.T) {
	manager := NewManager()

	// Create many timers
	for i := 0; i < 200; i++ {
		manager.CreateTimer(0.01, true, true, func() {})
	}

	// Update should reuse pre-allocated slices
	initialCallbacksCap := cap(manager.callbacks)
	initialCleanupCap := cap(manager.cleanupIDs)

	manager.Update(0.01)

	// Slices should remain allocated (or grow minimally)
	finalCallbacksCap := cap(manager.callbacks)
	finalCleanupCap := cap(manager.cleanupIDs)

	if finalCallbacksCap < initialCallbacksCap {
		t.Fatalf("callbacks slice capacity should not shrink: %d -> %d", initialCallbacksCap, finalCallbacksCap)
	}
	if finalCleanupCap < initialCleanupCap {
		t.Fatalf("cleanupIDs slice capacity should not shrink: %d -> %d", initialCleanupCap, finalCleanupCap)
	}
}

func TestTimer_ConcurrentUpdateAndCancel(t *testing.T) {
	timer := NewTimer(1, 1.0, true, true, func() {})
	done := make(chan bool, 2)

	// Update in one goroutine
	go func() {
		for i := 0; i < 100; i++ {
			timer.Update(0.001)
		}
		done <- true
	}()

	// Cancel in another goroutine
	go func() {
		for i := 0; i < 50; i++ {
			timer.Cancel()
		}
		done <- true
	}()

	<-done
	<-done

	if !timer.IsCancelled() {
		t.Fatal("timer should be cancelled")
	}
}

func TestTimerManager_RemoveWithCancelCallback(t *testing.T) {
	manager := NewManager()
	cancelCalled := 0

	timer := manager.CreateTimer(1.0, true, true, func() {})
	timer.SetCancelFunc(func() {
		cancelCalled++
	})

	manager.RemoveTimer(timer.ID())

	if cancelCalled != 1 {
		t.Fatalf("cancel callback should fire when timer is removed, got %d calls", cancelCalled)
	}
}

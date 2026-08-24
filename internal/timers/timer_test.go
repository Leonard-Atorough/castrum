package timers

import "testing"

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

	if fired := timer.Update(0.1); fired {
		t.Fatal("timer should not fire before duration completes")
	}
	if callbackCalled != 0 {
		t.Fatalf("callback should not fire before expiration, got %d calls", callbackCalled)
	}

	if fired := timer.Update(0.2); !fired {
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

	if fired := timer.Update(0.1); !fired {
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

	if fired := timer.Update(0.1); !fired {
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

	if fired := timer.Update(0.5); fired {
		t.Fatal("stopped timer should not fire")
	}
	if timer.Elapsed() != 0 {
		t.Fatalf("stopped timer should not accumulate elapsed while inactive, got %f", timer.Elapsed())
	}
}

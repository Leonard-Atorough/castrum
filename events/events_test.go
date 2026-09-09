package events

import (
	"sync"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	if bus == nil {
		t.Fatal("NewEventBus returned nil")
	}
	if bus.nextSubID != 1 {
		t.Errorf("expected nextSubID to be 1, got %d", bus.nextSubID)
	}
	if len(bus.subscribers) != 0 {
		t.Errorf("expected subscribers to be empty, got %d", len(bus.subscribers))
	}
}

func TestOn_SingleHandler(t *testing.T) {
	bus := NewEventBus()
	called := false

	subID := bus.On(func(meta EventMeta, e TestEvent) {
		called = true
	}, false)

	if subID != 1 {
		t.Errorf("expected subID 1, got %d", subID)
	}

	bus.Emit(TestEvent{value: 42}, "test")
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestOn_MultipleHandlers(t *testing.T) {
	bus := NewEventBus()
	call1, call2 := false, false

	bus.On(func(meta EventMeta, e TestEvent) {
		call1 = true
	}, false)
	bus.On(func(meta EventMeta, e TestEvent) {
		call2 = true
	}, false)

	bus.Emit(TestEvent{value: 42}, "test")
	if !call1 || !call2 {
		t.Fatal("not all handlers were called")
	}
}

func TestOn_ReturnsUniqueIDs(t *testing.T) {
	bus := NewEventBus()
	id1 := bus.On(func(meta EventMeta, e TestEvent) {}, false)
	id2 := bus.On(func(meta EventMeta, e TestEvent) {}, false)
	id3 := bus.On(func(meta EventMeta, e TestEvent) {}, false)

	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Fatalf("expected unique IDs, got %d, %d, %d", id1, id2, id3)
	}
}

func TestOn_MetadataPassedCorrectly(t *testing.T) {
	bus := NewEventBus()
	var capturedMeta EventMeta

	bus.On(func(meta EventMeta, e TestEvent) {
		capturedMeta = meta
	}, false)

	event := TestEvent{value: 99}
	bus.Emit(event, "mySource")

	if capturedMeta.Source != "mySource" {
		t.Errorf("expected source 'mySource', got '%s'", capturedMeta.Source)
	}
	if capturedMeta.EventType == "" {
		t.Fatal("EventType was empty")
	}
	if capturedMeta.Timestamp == 0 {
		t.Fatal("Timestamp was zero")
	}
}

func TestOn_EventDataPassedCorrectly(t *testing.T) {
	bus := NewEventBus()
	var capturedEvent TestEvent

	bus.On(func(meta EventMeta, e TestEvent) {
		capturedEvent = e
	}, false)

	event := TestEvent{value: 123}
	bus.Emit(event, "test")

	if capturedEvent.value != 123 {
		t.Errorf("expected value 123, got %d", capturedEvent.value)
	}
}

func TestOnce(t *testing.T) {
	bus := NewEventBus()
	callCount := 0

	bus.Once(func(meta EventMeta, e TestEvent) {
		callCount++
	})

	bus.Emit(TestEvent{value: 1}, "test")
	bus.Emit(TestEvent{value: 2}, "test")
	bus.Emit(TestEvent{value: 3}, "test")

	if callCount != 1 {
		t.Errorf("expected handler to be called once, was called %d times", callCount)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewEventBus()
	called := false

	subID := bus.On(func(meta EventMeta, e TestEvent) {
		called = true
	}, false)

	bus.Unsubscribe(subID)
	bus.Emit(TestEvent{value: 42}, "test")

	if called {
		t.Fatal("handler was called after unsubscribe")
	}
}

func TestUnsubscribe_NonExistent(t *testing.T) {
	bus := NewEventBus()
	bus.Unsubscribe(9999) // Should not crash
}

func TestOffAll(t *testing.T) {
	bus := NewEventBus()
	call1, call2 := false, false

	bus.On(func(meta EventMeta, e TestEvent) {
		call1 = true
	}, false)
	bus.On(func(meta EventMeta, e TestEvent) {
		call2 = true
	}, false)

	bus.OffAll[TestEvent]()
	bus.Emit(TestEvent{value: 42}, "test")

	if call1 || call2 {
		t.Fatal("handlers were called after OffAll")
	}
}

func TestClear(t *testing.T) {
	bus := NewEventBus()
	call1, call2 := false, false

	bus.On(func(meta EventMeta, e TestEvent) {
		call1 = true
	}, false)
	bus.On(func(meta EventMeta, e AnotherEvent) {
		call2 = true
	}, false)

	bus.Clear()

	bus.Emit(TestEvent{value: 42}, "test")
	bus.Emit(AnotherEvent{name: "test"}, "test")

	if call1 || call2 {
		t.Fatal("handlers were called after Clear")
	}
}

func TestHasSubscribers(t *testing.T) {
	bus := NewEventBus()

	if bus.HasSubscribers[TestEvent]() {
		t.Fatal("expected no subscribers initially")
	}

	bus.On(func(meta EventMeta, e TestEvent) {}, false)

	if !bus.HasSubscribers[TestEvent]() {
		t.Fatal("expected HasSubscribers to return true")
	}

	bus.OffAll[TestEvent]()

	if bus.HasSubscribers[TestEvent]() {
		t.Fatal("expected HasSubscribers to return false after OffAll")
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := NewEventBus()

	if bus.SubscriberCount[TestEvent]() != 0 {
		t.Fatal("expected count 0 initially")
	}

	bus.On(func(meta EventMeta, e TestEvent) {}, false)
	if bus.SubscriberCount[TestEvent]() != 1 {
		t.Errorf("expected count 1, got %d", bus.SubscriberCount[TestEvent]())
	}

	bus.On(func(meta EventMeta, e TestEvent) {}, false)
	if bus.SubscriberCount[TestEvent]() != 2 {
		t.Errorf("expected count 2, got %d", bus.SubscriberCount[TestEvent]())
	}

	bus.OffAll[TestEvent]()
	if bus.SubscriberCount[TestEvent]() != 0 {
		t.Errorf("expected count 0 after OffAll, got %d", bus.SubscriberCount[TestEvent]())
	}
}

func TestHandlerPanicDoesNotCrashBus(t *testing.T) {
	bus := NewEventBus()
	recovered := false
	secondHandlerCalled := false

	bus.On(func(meta EventMeta, e TestEvent) {
		panic("handler panic")
	}, false)

	bus.On(func(meta EventMeta, e TestEvent) {
		secondHandlerCalled = true
	}, false)

	// This should not panic or crash
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		bus.Emit(TestEvent{value: 42}, "test")
	}()

	if recovered {
		t.Fatal("panic propagated from bus.Emit")
	}
	if !secondHandlerCalled {
		t.Fatal("second handler was not called after first handler panicked")
	}
}

func TestHandlerCanEmitEvents_NoDeadlock(t *testing.T) {
	bus := NewEventBus()
	var mu sync.Mutex
	eventChain := []string{}

	bus.On(func(meta EventMeta, e TestEvent) {
		mu.Lock()
		eventChain = append(eventChain, "first")
		mu.Unlock()
		// Emit a different event from within handler
		bus.Emit(AnotherEvent{name: "chained"}, "test")
	}, false)

	bus.On(func(meta EventMeta, e AnotherEvent) {
		mu.Lock()
		eventChain = append(eventChain, "second")
		mu.Unlock()
	}, false)

	done := make(chan bool, 1)
	go func() {
		bus.Emit(TestEvent{value: 1}, "test")
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("deadlock detected: handler emit blocked")
	}

	if len(eventChain) != 2 || eventChain[0] != "first" || eventChain[1] != "second" {
		t.Errorf("expected event chain [first, second], got %v", eventChain)
	}
}

func TestDifferentEventTypes_Isolated(t *testing.T) {
	bus := NewEventBus()
	testCalled := false
	anotherCalled := false

	bus.On(func(meta EventMeta, e TestEvent) {
		testCalled = true
	}, false)

	bus.On(func(meta EventMeta, e AnotherEvent) {
		anotherCalled = true
	}, false)

	bus.Emit(TestEvent{value: 1}, "test")

	if !testCalled || anotherCalled {
		t.Fatal("event isolation failed")
	}

	testCalled = false
	bus.Emit(AnotherEvent{name: "test"}, "test")

	if testCalled || !anotherCalled {
		t.Fatal("event isolation failed for second event type")
	}
}

func TestConcurrentSubscriptions(t *testing.T) {
	bus := NewEventBus()
	var mu sync.Mutex
	subscriptionCount := 0

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			bus.On(func(meta EventMeta, e TestEvent) {
				mu.Lock()
				subscriptionCount++
				mu.Unlock()
			}, false)
		}(i)
	}

	wg.Wait()

	if bus.SubscriberCount[TestEvent]() != 100 {
		t.Errorf("expected 100 subscribers, got %d", bus.SubscriberCount[TestEvent]())
	}

	bus.Emit(TestEvent{value: 1}, "test")

	if subscriptionCount != 100 {
		t.Errorf("expected 100 handler calls, got %d", subscriptionCount)
	}
}

func TestConcurrentEmit(t *testing.T) {
	bus := NewEventBus()
	var mu sync.Mutex
	emitCount := 0

	bus.On(func(meta EventMeta, e TestEvent) {
		mu.Lock()
		emitCount++
		mu.Unlock()
	}, false)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			bus.Emit(TestEvent{value: index}, "test")
		}(i)
	}

	wg.Wait()

	if emitCount != 50 {
		t.Errorf("expected 50 emits, got %d", emitCount)
	}
}

func TestOnce_ConcurrentRemoval(t *testing.T) {
	bus := NewEventBus()
	var mu sync.Mutex
	callCount := 0

	bus.Once(func(meta EventMeta, e TestEvent) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Emit(TestEvent{value: 1}, "test")
		}()
	}

	wg.Wait()

	if callCount != 1 {
		t.Errorf("expected Once handler to be called exactly once, was called %d times", callCount)
	}
}

// Test event types
type TestEvent struct {
	value int
}

type AnotherEvent struct {
	name string
}

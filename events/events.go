package events

import (
	"reflect"
	"sync"
	"time"
)

type subscription struct {
	id      int
	handler func(EventMeta, any)
	once    bool
	fired   bool
}

type subscribers struct {
	eventType     reflect.Type
	subscriptions []subscription
}

type EventMeta struct {
	EventType string
	Timestamp int64
	Source    string
}

type EventBus struct {
	subscribers []subscribers
	nextSubID   int
	mu          sync.Mutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]subscribers, 0, 16),
		nextSubID:   1,
	}
}

func (eb *EventBus) On[T any](handler func(EventMeta, T), fireOnce bool) int {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subID := eb.nextSubID
	eb.nextSubID++

	typ := reflect.TypeFor[T]()
	wrappedHandler := func(ctx EventMeta, event any) {
		defer func() {
			if r := recover(); r != nil {
				// Handler panicked, but don't crash the bus
			}
		}()
		handler(ctx, event.(T))
	}

	for i := range eb.subscribers {
		if eb.subscribers[i].eventType == typ {
			eb.subscribers[i].subscriptions = append(eb.subscribers[i].subscriptions, subscription{
				id:      subID,
				handler: wrappedHandler,
				once:    fireOnce,
			})
			return subID
		}
	}

	eb.subscribers = append(eb.subscribers, subscribers{
		eventType: typ,
		subscriptions: []subscription{
			{
				id:      subID,
				handler: wrappedHandler,
				once:    fireOnce,
			},
		},
	})
	return subID
}

func (eb *EventBus) Once[T any](handler func(EventMeta, T)) int {
	return eb.On(handler, true)
}

func (eb *EventBus) Emit[T any](event T, source string) {
	eb.mu.Lock()
	typ := reflect.TypeFor[T]()
	var toCall []subscription

	for i := range eb.subscribers {
		if eb.subscribers[i].eventType != typ {
			continue
		}

		for j := range eb.subscribers[i].subscriptions {
			sub := &eb.subscribers[i].subscriptions[j]
			if sub.once && sub.fired {
				continue
			}
			if sub.once {
				sub.fired = true
			}
			toCall = append(toCall, *sub)
		}
		break
	}
	eb.mu.Unlock()

	if len(toCall) == 0 {
		return
	}

	ctx := EventMeta{
		EventType: typ.String(),
		Timestamp: time.Now().UnixNano(),
		Source:    source,
	}

	for _, s := range toCall {
		s.handler(ctx, event)
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	for i := range eb.subscribers {
		if eb.subscribers[i].eventType != typ {
			continue
		}
		subs := eb.subscribers[i].subscriptions[:0]
		for _, sub := range eb.subscribers[i].subscriptions {
			if sub.once && sub.fired {
				continue
			}
			subs = append(subs, sub)
		}
		eb.subscribers[i].subscriptions = subs
		break
	}
}

func (eb *EventBus) Unsubscribe(subID int) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for i := range eb.subscribers {
		for j := 0; j < len(eb.subscribers[i].subscriptions); j++ {
			if eb.subscribers[i].subscriptions[j].id == subID {
				eb.subscribers[i].subscriptions = append(
					eb.subscribers[i].subscriptions[:j],
					eb.subscribers[i].subscriptions[j+1:]...,
				)
				j--
				return
			}
		}
	}
}

func (eb *EventBus) OffAll[T any]() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	typ := reflect.TypeFor[T]()
	for i := 0; i < len(eb.subscribers); i++ {
		if eb.subscribers[i].eventType == typ {
			eb.subscribers = append(eb.subscribers[:i], eb.subscribers[i+1:]...)
			i--
		}
	}
}

func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers = make([]subscribers, 0, 16)
	eb.nextSubID = 1
}

func (eb *EventBus) HasSubscribers[T any]() bool {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	typ := reflect.TypeFor[T]()
	for _, sub := range eb.subscribers {
		if sub.eventType == typ && len(sub.subscriptions) > 0 {
			return true
		}
	}
	return false
}

func (eb *EventBus) SubscriberCount[T any]() int {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	typ := reflect.TypeFor[T]()
	for _, sub := range eb.subscribers {
		if sub.eventType == typ {
			return len(sub.subscriptions)
		}
	}
	return 0
}

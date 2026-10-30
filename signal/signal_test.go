package signal

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

type ping struct {
	n int
}

type pong struct {
	n int
}

// tickHub returns a hub positioned where the engine enters the fixed loop:
// the frame phase has ended, mode is tick, and both windows are empty.
func tickHub() *Hub {
	h := NewHub()
	h.BeginFrame()
	h.EndFrame()
	return h
}

func collect[T any](t *testing.T, r *Reader[T]) []T {
	t.Helper()
	var got []T
	for value, ok := r.Next(); ok; value, ok = r.Next() {
		got = append(got, value)
	}
	return got
}

func want[T any](t *testing.T, name string, got, expected []T) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("%s: got %v, want %v", name, got, expected)
	}
}

func TestTickSignalsDeliveredOnceInOrder(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})
	h.Emit(ping{2})

	r := h.NewReader[ping]()
	want(t, "first read", collect(t, r), []ping{{1}, {2}})
	want(t, "second read", collect(t, r), nil)
}

func TestReaderIndependence(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})
	h.Emit(ping{2})

	a := h.NewReader[ping]()
	b := h.NewReader[ping]()
	want(t, "reader a", collect(t, a), []ping{{1}, {2}})
	want(t, "reader b", collect(t, b), []ping{{1}, {2}})
	want(t, "reader a again", collect(t, a), nil)
}

func TestReaderAdoptsWindowInProgress(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})

	r := h.NewReader[ping]()
	want(t, "late reader", collect(t, r), []ping{{1}})
}

func TestEmitDuringReadDefersUntilNextPass(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()
	h.Emit(ping{1})

	var seen []ping
	value, ok := r.Next()
	if !ok {
		t.Fatal("first Next returned no signal")
	}
	seen = append(seen, value)
	h.Emit(ping{2})
	if _, ok := r.Next(); ok {
		t.Fatal("signal emitted during the read pass was returned in the same pass")
	}
	want(t, "mid-read emit deferred", seen, []ping{{1}})
	want(t, "next read pass in same window", collect(t, r), []ping{{2}})
}

func TestSignalReadInBothPhases(t *testing.T) {
	h := tickHub()
	tickReader := h.NewReader[ping]()

	h.Emit(ping{1})
	want(t, "tick phase", collect(t, tickReader), []ping{{1}})

	h.EndTick()
	h.BeginFrame()
	frameReader := h.NewReader[ping]()
	want(t, "frame phase", collect(t, frameReader), []ping{{1}})
}

func TestTickReaderSeesNoSignalsFromPreviousTick(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()

	h.Emit(ping{1})
	want(t, "tick 1", collect(t, r), []ping{{1}})

	h.EndTick()
	h.Emit(ping{2})
	want(t, "tick 2", collect(t, r), []ping{{2}})
}

func TestCursorResetsWhenWindowTurnsOver(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()

	h.Emit(ping{1})
	h.Emit(ping{2})
	want(t, "two signals", collect(t, r), []ping{{1}, {2}})

	h.EndTick()
	h.Emit(ping{3})
	want(t, "window reuse", collect(t, r), []ping{{3}})
}

func TestUnreadTickSignalsReachFrameReaders(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()
	h.Emit(ping{1})

	h.EndTick()
	want(t, "tick reader", collect(t, r), nil)

	h.BeginFrame()
	want(t, "frame reader", collect(t, h.NewReader[ping]()), []ping{{1}})
}

func TestFrameWindowHoldsAllTicksOfTheFrame(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})
	h.EndTick()
	h.Emit(ping{2})
	h.EndTick()

	h.BeginFrame()
	want(t, "frame window", collect(t, h.NewReader[ping]()), []ping{{1}, {2}})
}

func TestFrameSignalsDroppedAfterFramePhase(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})
	h.EndTick()

	h.BeginFrame()
	want(t, "frame 1 phase", collect(t, h.NewReader[ping]()), []ping{{1}})
	h.EndFrame()

	h.BeginFrame()
	want(t, "frame 2 phase, zero ticks", collect(t, h.NewReader[ping]()), nil)
}

// NewHub starts in frame mode, as during startup.
func TestFrameEmitsDieAtFrameEnd(t *testing.T) {
	h := NewHub()
	h.Emit(ping{1})
	want(t, "same frame phase", collect(t, h.NewReader[ping]()), []ping{{1}})

	h.EndFrame()
	h.BeginFrame()
	want(t, "next frame phase", collect(t, h.NewReader[ping]()), nil)
}

func TestSignalTypesAreIsolated(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})
	h.Emit(pong{1})

	want(t, "ping reader", collect(t, h.NewReader[ping]()), []ping{{1}})
	want(t, "pong reader", collect(t, h.NewReader[pong]()), []pong{{1}})
}

func TestReaderConsumesSignalBeforeHandling(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()
	h.Emit(ping{1})
	h.Emit(ping{2})
	h.Emit(ping{3})

	value, ok := r.Next()
	if !ok || value != (ping{1}) {
		t.Fatalf("first signal = %v, %t; want ping 1", value, ok)
	}
	value, ok = r.Next()
	if !ok || value != (ping{2}) {
		t.Fatalf("second signal = %v, %t; want ping 2", value, ok)
	}
	// Handling value 2 may fail, but Next already advanced the cursor.
	want(t, "remaining signals", collect(t, r), []ping{{3}})
}

func TestConsumerOwnsItsReader(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})

	ctx := &core.Context{}
	var a, b []ping
	sysA := h.Consumer(func(ctx *core.Context, v ping) error { a = append(a, v); return nil })
	sysB := h.Consumer(func(ctx *core.Context, v ping) error { b = append(b, v); return nil })

	for _, sys := range []core.System{sysA, sysB} {
		if err := sys.Update(ctx); err != nil {
			t.Fatalf("consumer update: %v", err)
		}
		if err := sys.Update(ctx); err != nil {
			t.Fatalf("second update: %v", err)
		}
	}
	want(t, "consumer a", a, []ping{{1}})
	want(t, "consumer b", b, []ping{{1}})
}

func TestConsumerPropagatesHandlerError(t *testing.T) {
	h := tickHub()
	h.Emit(ping{1})
	h.Emit(ping{2})

	sentinel := errors.New("boom")
	var got []ping
	sys := h.Consumer(func(ctx *core.Context, v ping) error {
		got = append(got, v)
		if v.n == 1 {
			return sentinel
		}
		return nil
	})
	if err := sys.Update(&core.Context{}); !errors.Is(err, sentinel) {
		t.Fatalf("consumer error: got %v, want %v", err, sentinel)
	}
	if err := sys.Update(&core.Context{}); err != nil {
		t.Fatalf("consumer retry: %v", err)
	}
	want(t, "consumer signals across error", got, []ping{{1}, {2}})
}

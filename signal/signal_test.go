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
	if err := r.Drain(func(v T) error {
		got = append(got, v)
		return nil
	}); err != nil {
		t.Fatalf("drain: %v", err)
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
	want(t, "first drain", collect(t, r), []ping{{1}, {2}})
	want(t, "second drain", collect(t, r), nil)
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

func TestEmitDuringDrainDefers(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()
	h.Emit(ping{1})

	var seen []ping
	if err := r.Drain(func(v ping) error {
		seen = append(seen, v)
		h.Emit(ping{2})
		return nil
	}); err != nil {
		t.Fatalf("drain: %v", err)
	}
	want(t, "mid-drain emit deferred", seen, []ping{{1}})
	want(t, "next drain in same window", collect(t, r), []ping{{2}})
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

func TestUndrainedTickSignalsReachFrameReaders(t *testing.T) {
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

func TestDrainStopsAtHandlerError(t *testing.T) {
	h := tickHub()
	r := h.NewReader[ping]()
	h.Emit(ping{1})
	h.Emit(ping{2})
	h.Emit(ping{3})

	sentinel := errors.New("boom")
	err := r.Drain(func(v ping) error {
		if v.n == 2 {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("drain error: got %v, want %v", err, sentinel)
	}
	want(t, "resume after error", collect(t, r), []ping{{3}})
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

	sentinel := errors.New("boom")
	sys := h.Consumer(func(ctx *core.Context, v ping) error { return sentinel })
	if err := sys.Update(&core.Context{}); !errors.Is(err, sentinel) {
		t.Fatalf("consumer error: got %v, want %v", err, sentinel)
	}
}

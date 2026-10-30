// Package signal provides typed, pull-based broadcasts for systems. A [Hub]
// stores signals by type, and each [Reader] maintains an independent cursor.
// Tick signals are readable during their tick and carried into the following
// frame window; frame signals are discarded at [Hub.EndFrame].
package signal

import (
	"reflect"

	"github.com/Leonard-Atorough/castrum/core"
)

type mode uint8

const (
	modeTick mode = iota
	modeFrame
)

// window is the non-generic boundary: each map entry holds a *buffer[T] for
// the type it is keyed by, and the Hub rotates every buffer through this
// interface without knowing T.
type window interface {
	flushTick()
	clearFrame()
}

type buffer[T any] struct {
	tick, frame       []T
	tickGen, frameGen uint32
}

func (b *buffer[T]) flushTick() {
	b.frame = append(b.frame, b.tick...)
	b.tick = b.tick[:0]
	b.tickGen++
}

func (b *buffer[T]) clearFrame() {
	b.frame = b.frame[:0]
	b.frameGen++
}

// Hub stores signals by type and rotates them between frame and tick windows.
type Hub struct {
	buffers map[reflect.Type]window
	mode    mode
}

// NewHub creates a Hub ready to collect frame signals.
func NewHub() *Hub {
	return &Hub{
		buffers: make(map[reflect.Type]window),
		mode:    modeFrame,
	}
}

// Emit adds value to the current frame or tick window.
func (h *Hub) Emit[T any](value T) {
	b := h.bufferFor[T]()
	if h.mode == modeTick {
		b.tick = append(b.tick, value)
	} else {
		b.frame = append(b.frame, value)
	}
}

// NewReader returns an independent reader for signals of type T. It starts
// at the beginning of the current window, so it can read signals already
// emitted in that window.
func (h *Hub) NewReader[T any]() *Reader[T] {
	buf := h.bufferFor[T]()
	return &Reader[T]{
		hub:      h,
		buf:      buf,
		tickGen:  buf.tickGen,
		frameGen: buf.frameGen,
	}
}

// Consumer returns a system that calls update for each unread signal of type
// T in the current window. Each returned system has an independent [Reader],
// so consumers do not compete for signals.
//
// Use [Hub.NewReader] when one system reads several signal types or needs to
// manage the reader's lifetime directly.
func (h *Hub) Consumer[T any](update func(ctx *core.Context, value T) error) core.System {
	reader := h.NewReader[T]()
	return core.SystemFunc(func(ctx *core.Context) error {
		for value, ok := reader.Next(); ok; value, ok = reader.Next() {
			if err := update(ctx, value); err != nil {
				return err
			}
		}
		return nil
	})
}

// BeginFrame starts collecting frame signals.
func (h *Hub) BeginFrame() {
	h.mode = modeFrame
}

// EndFrame discards remaining frame signals and starts tick collection.
func (h *Hub) EndFrame() {
	for _, w := range h.buffers {
		w.clearFrame()
	}
	h.mode = modeTick
}

// EndTick moves tick signals into the frame window for readers to read.
func (h *Hub) EndTick() {
	for _, w := range h.buffers {
		w.flushTick()
	}
}

func (h *Hub) bufferFor[T any]() *buffer[T] {
	typ := reflect.TypeFor[T]()
	w, ok := h.buffers[typ]
	var b *buffer[T]
	if ok {
		b = w.(*buffer[T])
	} else {
		b = &buffer[T]{}
		h.buffers[typ] = b
	}
	return b
}

// Reader tracks an independent position while reading signals of type T.
// Sharing a Reader between consumers splits delivery between them.
type Reader[T any] struct {
	hub                   *Hub
	buf                   *buffer[T]
	tickGen, frameGen     uint32
	tickIdx, frameIdx     int
	tickEnd, frameEnd     int
	tickBatch, frameBatch bool
}

// Next returns the next unread signal in the current window. The signal is
// consumed when returned. It reports false when the current batch is exhausted;
// signals emitted during that batch are available on the next read pass.
func (r *Reader[T]) Next() (T, bool) {
	if r.hub.mode == modeTick {
		if r.tickGen != r.buf.tickGen {
			r.tickGen = r.buf.tickGen
			r.tickIdx = 0
			r.tickBatch = false
		}
		if !r.tickBatch {
			r.tickEnd = len(r.buf.tick)
			r.tickBatch = true
		}
		if r.tickIdx < r.tickEnd {
			value := r.buf.tick[r.tickIdx]
			r.tickIdx++
			return value, true
		}
		r.tickBatch = false
		var zero T
		return zero, false
	}
	if r.frameGen != r.buf.frameGen {
		r.frameGen = r.buf.frameGen
		r.frameIdx = 0
		r.frameBatch = false
	}
	if !r.frameBatch {
		r.frameEnd = len(r.buf.frame)
		r.frameBatch = true
	}
	if r.frameIdx < r.frameEnd {
		value := r.buf.frame[r.frameIdx]
		r.frameIdx++
		return value, true
	}
	r.frameBatch = false
	var zero T
	return zero, false
}

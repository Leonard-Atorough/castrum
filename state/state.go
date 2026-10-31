// Package state provides generic state machines whose transitions are driven
// by scheduled system updates.
package state

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
)

// ErrTransitionPending reports that a machine already has a different
// transition request waiting to be processed.
var ErrTransitionPending = errors.New("state transition already pending")

// Initialized is a signal emitted the first time a machine's initial enter
// hooks have run successfully. It contains the initial state whose enter hooks
// completed.
type Initialized[S comparable] struct {
	// State is the initial state whose enter hooks completed.
	State S
}

// Changed is a signal emitted when a transition or restart is applied.
// It is emitted after exit hooks succeed and before enter hooks run. This
// means that a Changed event guarantees that the state has changed but not
// that enter hooks have run successfully.
type Changed[S comparable] struct {
	// From is the state the machine left.
	From S
	// To is the machine's new current state.
	To S
}

// Machine tracks a comparable state value and queues requested transitions
// until its system processes them. Its zero value starts in S's zero state.
// Access a machine only from the game loop's goroutine.
type Machine[S comparable] struct {
	current        S
	pending        S
	hasPending     bool
	pendingRestart bool
	initialized    bool
	entryHooks     map[S][]core.SystemFunc
	exitHooks      map[S][]core.SystemFunc
}

// New returns a machine starting in S's zero state.
func New[S comparable]() *Machine[S] {
	return &Machine[S]{
		entryHooks: make(map[S][]core.SystemFunc),
		exitHooks:  make(map[S][]core.SystemFunc),
	}
}

// WithInitial returns a new machine starting in initial, with a copy of m's
// registered hooks. It leaves m unchanged and carries over neither pending
// requests nor initialization progress.
func (m *Machine[S]) WithInitial(initial S) *Machine[S] {
	copyHooks := func(hooks map[S][]core.SystemFunc) map[S][]core.SystemFunc {
		result := maps.Clone(hooks)
		for value, entries := range result {
			result[value] = slices.Clone(entries)
		}
		return result
	}
	return &Machine[S]{
		current:    initial,
		pending:    initial,
		entryHooks: copyHooks(m.entryHooks),
		exitHooks:  copyHooks(m.exitHooks),
	}
}

// Current returns the machine's current state.
func (m *Machine[S]) Current() S {
	return m.current
}

// Is reports whether value is the current state.
func (m *Machine[S]) Is(value S) bool {
	return m.current == value
}

// Pending returns the accepted destination waiting to be processed, if any.
func (m *Machine[S]) Pending() (S, bool) {
	return m.pending, m.hasPending
}

// OnEnter registers a [core.SystemFunc] that runs when value is entered,
// including the initial state during startup. An error stops the remaining
// hooks and is returned by the state system. A nil hook returns an error when
// run.
func (m *Machine[S]) OnEnter(value S, hook core.SystemFunc) {
	if m.entryHooks == nil {
		m.entryHooks = make(map[S][]core.SystemFunc)
	}
	m.entryHooks[value] = append(m.entryHooks[value], hook)
}

// OnExit registers a [core.SystemFunc] that runs before leaving value. An error
// stops the transition and is returned by the state system. A nil hook returns
// an error when run.
func (m *Machine[S]) OnExit(value S, hook core.SystemFunc) {
	if m.exitHooks == nil {
		m.exitHooks = make(map[S][]core.SystemFunc)
	}
	m.exitHooks[value] = append(m.exitHooks[value], hook)
}

// Request accepts next as the pending destination without running hooks.
// Requests for the current state are no-ops; an identical pending request
// coalesces, while a different pending destination returns
// [ErrTransitionPending]. A successful request means it was accepted, not
// that the transition completed.
func (m *Machine[S]) Request(next S) error {
	if m.current == next {
		return nil
	}
	if m.hasPending {
		if m.pending == next {
			return nil
		}
		return fmt.Errorf("state: %w: pending %v, requested %v", ErrTransitionPending, m.pending, next)
	}
	m.pending = next
	m.hasPending = true
	m.pendingRestart = false
	return nil
}

// Restart accepts a request to exit and re-enter the current state on the
// next update. Repeated restart requests coalesce. If another destination is
// already pending, Restart returns an error wrapping [ErrTransitionPending].
// A successful request means it was accepted, not that the restart completed.
func (m *Machine[S]) Restart() error {
	if m.hasPending {
		if m.pending == m.current {
			return nil
		}
		return fmt.Errorf("state: %w: pending %v", ErrTransitionPending, m.pending)
	}
	m.pending = m.current
	m.hasPending = true
	m.pendingRestart = true
	return nil
}

func (m *Machine[S]) apply(ctx *core.Context, hub *signal.Hub) error {
	if !m.initialized {
		m.initialized = true
		if err := runHooks("enter", m.current, m.entryHooks[m.current], ctx); err != nil {
			return err
		}
		hub.Emit(Initialized[S]{State: m.current})
		return nil
	}
	if !m.hasPending {
		return nil
	}
	next := m.pending
	m.hasPending = false
	m.pendingRestart = false
	if err := runHooks("exit", m.current, m.exitHooks[m.current], ctx); err != nil {
		return err
	}
	previous := m.current
	m.current = next
	if m.hasPending && m.pending == m.current && !m.pendingRestart {
		m.hasPending = false
	}
	hub.Emit(Changed[S]{From: previous, To: next})
	return runHooks("enter", m.current, m.entryHooks[m.current], ctx)
}

func runHooks[S comparable](action string, value S, hooks []core.SystemFunc, ctx *core.Context) error {
	for _, hook := range hooks {
		if hook == nil {
			return fmt.Errorf("state: %s hook for %v: hook must not be nil", action, value)
		}
		if err := hook(ctx); err != nil {
			return fmt.Errorf("state: %s hook for %v: %w", action, value, err)
		}
	}
	return nil
}

// SystemName is the registered name for the state system.
const SystemName = "state"

// NewSystem returns a [core.System] that processes a [Machine]'s initial entry
// and queued transitions.
//
// Register it in [core.PhasePreStartup] and [core.PhasePreFrame] to run initial
// entry before startup scene operations and transitions before frame scene
// operations. The first update runs the initial state's enter hooks; later
// updates apply pending transitions or restarts. Requests accepted from hooks
// are deferred until the next update.
//
// The system resolves and caches the world's [signal.Hub] before processing
// the machine. Resolution errors leave the machine unchanged. It emits
// [Initialized] after successful initial entry and [Changed] after exit hooks
// succeed, before enter hooks run. In the engine, these signals expire before
// fixed ticks.
//
// Hook errors stop the remaining hooks for that run and are returned from
// Update. An exit-hook error consumes the pending request without changing the
// current state. An enter-hook error during a transition is returned after
// [Changed] is emitted and the state has changed. Failed hooks are not retried,
// and their side effects are not rolled back. A failed initial entry does not
// emit [Initialized]. A nil machine returns an error from Update.
func NewSystem[S comparable](machine *Machine[S]) core.System {
	var hub *signal.Hub
	return core.SystemFunc(func(ctx *core.Context) error {
		if machine == nil {
			return errors.New("state: machine must not be nil")
		}
		if hub == nil {
			if ctx == nil || ctx.World == nil {
				return errors.New("state system: resolve signal hub: context and world must not be nil")
			}
			resolved, err := ctx.World.Resource[*signal.Hub]()
			if err != nil {
				return fmt.Errorf("state system: resolve signal hub: %w; provide *signal.Hub to the world through World.Provide, or construct the game with castrum.New", err)
			}
			if resolved == nil {
				return errors.New("state system: resolve signal hub: provided *signal.Hub must not be nil")
			}
			hub = resolved
		}
		return machine.apply(ctx, hub)
	})
}

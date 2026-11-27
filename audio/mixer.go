package audio

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/Leonard-Atorough/castrum/tween"
)

// BusName identifies a mixer bus.
type BusName string

// Mixer holds master and bus volume and mute state, and global and bus pause state.
// Its zero value has a master volume of zero; use [NewMixer] for unity.
type Mixer struct {
	main   bus
	buses  map[BusName]*bus
	paused bool
}

type bus struct {
	volume float64
	muted  bool
	paused bool
	fade   *fadeState
}

type fadeState struct {
	from     float64
	to       float64
	playhead tween.Tween
}

func newFade(from, to float64, over time.Duration) (*fadeState, error) {
	if math.IsNaN(to) {
		return nil, fmt.Errorf("audio: mixer: fade target volume cannot be NaN")
	}

	playhead, err := tween.NewTween(over, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: mixer: failed to create fade tween: %w", err)
	}

	return &fadeState{
		from:     from,
		to:       clamp01(to),
		playhead: playhead,
	}, nil
}

// NewMixer returns a mixer with unity master volume and no configured buses.
func NewMixer() *Mixer {
	return &Mixer{
		main: bus{
			volume: 1,
		},
		buses: make(map[BusName]*bus),
	}
}

// SetMasterVolume sets the master volume, clamped to [0, 1].
// It cancels any master fade and returns an error for NaN without changing the volume or fade.
func (m *Mixer) SetMasterVolume(volume float64) error {
	if math.IsNaN(volume) {
		return fmt.Errorf("audio: mixer: master volume cannot be NaN")
	}
	m.main.volume = clamp01(volume)
	m.main.fade = nil
	return nil
}

// MasterVolume returns the master volume, independently of mute state.
func (m *Mixer) MasterVolume() float64 {
	return m.main.volume
}

// ToggleMasterMuted toggles the master mute state.
func (m *Mixer) ToggleMasterMuted() {
	m.main.muted = !m.main.muted
}

// MasterMuted reports whether the master is muted.
func (m *Mixer) MasterMuted() bool {
	return m.main.muted
}

// FadeMaster starts a linear master volume fade to the target over the given duration.
// It replaces any active master fade and clamps the target to [0, 1].
// It returns an error for NaN or a non-positive duration without changing the fade.
func (m *Mixer) FadeMaster(to float64, over time.Duration) error {
	fade, err := newFade(m.main.volume, to, over)
	if err != nil {
		return err
	}
	m.main.fade = fade
	return nil
}

// SetBusVolume sets a bus's volume, clamped to [0, 1].
// It creates the bus if needed and cancels its fade. NaN returns an error
// without creating or changing a bus.
func (m *Mixer) SetBusVolume(name BusName, volume float64) error {
	if math.IsNaN(volume) {
		return fmt.Errorf("audio: mixer: bus volume cannot be NaN")
	}
	b := m.getOrCreateBus(name)
	b.volume = clamp01(volume)
	b.fade = nil
	return nil
}

// BusVolume returns a bus's volume, independently of mute state.
// An unconfigured bus has unity volume.
func (m *Mixer) BusVolume(name BusName) float64 {
	b, ok := m.buses[name]
	if !ok {
		return 1
	}
	return b.volume
}

// SetBusPaused sets a bus's local pause state, creating the bus if needed.
func (m *Mixer) SetBusPaused(name BusName, paused bool) {
	m.getOrCreateBus(name).paused = paused
}

// BusPaused reports whether a bus is locally paused or the mixer is globally paused.
func (m *Mixer) BusPaused(name BusName) bool {
	b, ok := m.buses[name]
	if !ok {
		return m.paused
	}
	return b.paused || m.paused
}

// SetBusMuted sets a bus's mute state, creating the bus if needed.
func (m *Mixer) SetBusMuted(name BusName, muted bool) {
	m.getOrCreateBus(name).muted = muted
}

// BusMuted reports whether a bus is muted, independently of master mute.
// An unconfigured bus is not muted.
func (m *Mixer) BusMuted(name BusName) bool {
	b, ok := m.buses[name]
	if !ok {
		return false
	}
	return b.muted
}

// RemoveBus removes a configured bus and its local state.
// It does nothing if the bus is not configured.
func (m *Mixer) RemoveBus(name BusName) {
	delete(m.buses, name)
}

// Fade starts a linear bus volume fade to the target over the given duration.
// It creates the bus if needed and replaces its active fade. The target is
// clamped to [0, 1]; NaN or a non-positive duration returns an error without
// creating or changing a bus.
func (m *Mixer) Fade(name BusName, to float64, over time.Duration) error {
	fade, err := newFade(m.BusVolume(name), to, over)
	if err != nil {
		return err
	}
	m.getOrCreateBus(name).fade = fade
	return nil
}

func (m *Mixer) update(delta time.Duration) error {
	if delta < 0 {
		return fmt.Errorf("audio: mixer: fade delta must not be negative")
	}
	if err := m.main.advanceFade(delta); err != nil {
		return fmt.Errorf("audio: mixer: advance master fade: %w", err)
	}
	for name, b := range m.buses {
		if err := b.advanceFade(delta); err != nil {
			return fmt.Errorf("audio: mixer: advance bus %q fade: %w", name, err)
		}
	}
	return nil
}

func (b *bus) advanceFade(delta time.Duration) error {
	if b.fade == nil {
		return nil
	}
	progress, err := b.fade.playhead.Update(delta)
	if err != nil {
		return err
	}
	b.volume = b.fade.from + (b.fade.to-b.fade.from)*progress
	if b.fade.playhead.IsComplete() {
		b.volume = b.fade.to
		b.fade = nil
	}
	return nil
}

// Buses returns the configured bus names in sorted order.
func (m *Mixer) Buses() []BusName {
	keys := make([]BusName, 0, len(m.buses))
	for k := range m.buses {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// HasBus reports whether a bus is configured.
func (m *Mixer) HasBus(name BusName) bool {
	_, ok := m.buses[name]
	return ok
}

// PauseAll sets the global pause state.
func (m *Mixer) PauseAll() {
	m.paused = true
}

// ResumeAll clears the global pause state without changing local bus pauses.
func (m *Mixer) ResumeAll() {
	m.paused = false
}

// AllPaused reports whether the mixer is globally paused.
func (m *Mixer) AllPaused() bool {
	return m.paused
}

// ToggleAllPaused toggles the global pause state.
func (m *Mixer) ToggleAllPaused() {
	m.paused = !m.paused
}

func (m *Mixer) getOrCreateBus(name BusName) *bus {
	if m.buses == nil {
		m.buses = make(map[BusName]*bus)
	}

	b, ok := m.buses[name]
	if !ok {
		b = &bus{volume: 1}
		m.buses[name] = b
	}
	return b
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

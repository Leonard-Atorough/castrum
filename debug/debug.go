// Package debug provides the development panel behind the debug overlay:
// systems append short-lived diagnostic text through a world resource, and
// the overlay draws the most recent messages beneath the frame and tick
// rates.
package debug

// Builtin identifies a measured section of the debug overlay.
type Builtin uint8

const (
	// FPS is the measured display frame rate.
	FPS Builtin = iota
	// TPS is the measured fixed tick rate.
	TPS

	// builtinCount is the number of builtins above; it guards the toggle
	// bitmask and is not a section itself.
	builtinCount
)

// messageCapacity is how many messages the panel retains.
const messageCapacity = 8

// Panel is the development message panel, provided by castrum.New and read
// by the debug overlay. Systems call [Panel.Message] with already-formatted
// text; the overlay draws the most recent messages below the rate line.
//
// The panel timestamps nothing and keeps no history beyond the last
// messageCapacity messages - format tick or frame numbers into the text
// when they matter. Like the world, the panel is loop-owned: Message,
// Clear, and Toggle belong on the engine goroutine.
type Panel struct {
	messages [messageCapacity]string
	next     int
	count    int
	enabled  uint8
}

// NewPanel returns a panel with every builtin enabled and no messages.
func NewPanel() *Panel {
	return &Panel{enabled: 1<<builtinCount - 1}
}

// Message appends text to the panel. Once messageCapacity messages are
// retained, the oldest is dropped.
func (p *Panel) Message(text string) {
	p.messages[p.next] = text
	p.next = (p.next + 1) % messageCapacity
	if p.count < messageCapacity {
		p.count++
	}
}

// Clear removes every message.
func (p *Panel) Clear() {
	p.next, p.count = 0, 0
	for i := range p.messages {
		p.messages[i] = ""
	}
}

// Lines returns the retained messages, oldest first. The returned slice is
// fresh; the panel's storage is reused.
func (p *Panel) Lines() []string {
	lines := make([]string, 0, p.count)
	start := (p.next - p.count + messageCapacity) % messageCapacity
	for i := range p.count {
		lines = append(lines, p.messages[(start+i)%messageCapacity])
	}
	return lines
}

// Toggle flips whether the overlay draws builtin's section. A value outside
// the defined builtins leaves the panel unchanged.
func (p *Panel) Toggle(b Builtin) {
	if b >= builtinCount {
		return
	}
	p.enabled ^= 1 << b
}

// Enabled reports whether the overlay draws builtin's section. A value
// outside the defined builtins is always disabled.
func (p *Panel) Enabled(b Builtin) bool {
	return b < builtinCount && p.enabled&(1<<b) != 0
}

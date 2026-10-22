// Package debug provides the world-owned [Panel] used to show status
// readouts, recent messages, and built-in frame and tick rates in the
// development overlay.
package debug

// Builtin identifies a built-in rate readout in the debug overlay.
type Builtin uint8

const (
	// FPS is the measured display frame rate.
	FPS Builtin = iota
	// TPS is the measured fixed-tick rate.
	TPS

	// builtinCount is the number of builtins above; it guards the toggle
	// bitmask and is not a section itself.
	builtinCount
)

const messageCapacity = 8

// Panel stores status readouts and recent messages for the debug overlay.
//
// Castrum provides it as a world resource. Use [Panel.Status] for persistent
// readouts and [Panel.Message] for the bounded message log. Access the panel
// from the engine goroutine; it is not safe for concurrent use.
type Panel struct {
	messages [messageCapacity]string
	next     int
	count    int
	enabled  uint8

	statuses    []statusLine
	statusIndex map[string]int
}

type statusLine struct {
	name string
	text string
}

// NewPanel returns a panel with all built-in rate readouts enabled and no
// messages or status lines.
func NewPanel() *Panel {
	return &Panel{
		enabled:     1<<builtinCount - 1,
		statusIndex: make(map[string]int),
	}
}

// Message appends text to the message log, dropping the oldest message when
// the log is full.
func (p *Panel) Message(text string) {
	p.messages[p.next] = text
	p.next = (p.next + 1) % messageCapacity
	if p.count < messageCapacity {
		p.count++
	}
}

// Clear removes all messages and status slots without changing which built-in
// rate readouts are enabled.
func (p *Panel) Clear() {
	p.next, p.count = 0, 0
	for i := range p.messages {
		p.messages[i] = ""
	}
	p.statuses = p.statuses[:0]
	clear(p.statusIndex)
}

// Status updates a named readout above the message log, creating its slot on
// first write. Later writes replace the text in place without changing its
// order. Empty text hides the line but keeps its slot, so it can reappear in
// the same position. Write changing readouts each frame; unchanged text
// remains visible until updated or cleared.
func (p *Panel) Status(name, text string) {
	if i, ok := p.statusIndex[name]; ok {
		p.statuses[i].text = text
		return
	}
	p.statusIndex[name] = len(p.statuses)
	p.statuses = append(p.statuses, statusLine{name: name, text: text})
}

// StatusLines returns the non-empty status lines in first-write order.
// The returned slice is independent of the panel's storage.
func (p *Panel) StatusLines() []string {
	lines := make([]string, 0, len(p.statuses))
	for _, s := range p.statuses {
		if s.text != "" {
			lines = append(lines, s.text)
		}
	}
	return lines
}

// Lines returns retained messages from oldest to newest. The returned slice
// is independent of the panel's storage.
func (p *Panel) Lines() []string {
	lines := make([]string, 0, p.count)
	start := (p.next - p.count + messageCapacity) % messageCapacity
	for i := range p.count {
		lines = append(lines, p.messages[(start+i)%messageCapacity])
	}
	return lines
}

// Toggle changes whether the overlay draws b. Undefined values have no
// effect.
func (p *Panel) Toggle(b Builtin) {
	if b >= builtinCount {
		return
	}
	p.enabled ^= 1 << b
}

// Enabled reports whether b is enabled for display. Undefined values are
// always disabled.
func (p *Panel) Enabled(b Builtin) bool {
	return b < builtinCount && p.enabled&(1<<b) != 0
}

package timers

type TimerID uint64

type Timer struct {
	// Unique identifier for the timer
	id TimerID
	// Duration for which the timer runs
	duration float64
	// Elapsed time since the timer started
	elapsed float64
	// Indicates whether the timer is currently running
	running bool
	// Indicates whether the timer should start automatically upon creation
	autoStart bool
	// Indicates whether the timer is a fire once timer or a repeating timer
	once bool
	// optional callback function to be called when the timer completes
	timerFunc func()
}

func NewTimer(id TimerID, duration float64, once bool, autoStart bool, timerFunc func()) *Timer {
	return &Timer{
		id:        id,
		duration:  duration,
		elapsed:   0,
		running:   autoStart,
		autoStart: autoStart,
		once:      once,
		timerFunc: timerFunc,
	}
}

func (t *Timer) Start() {
	t.running = true
	t.elapsed = 0
}

func (t *Timer) Stop() {
	t.running = false
}

func (t *Timer) Resume() {
	t.running = true
}

func (t *Timer) Update(deltaTime float64) bool {
	if !t.running {
		return false
	}

	t.elapsed += deltaTime

	if t.elapsed < t.duration {
		return false
	}

	t.running = false

	if !t.once {
		t.elapsed = 0
		t.running = true
	}

	return true
}

func (t *Timer) AutoStart() bool {
	return t.autoStart
}

func (t *Timer) IsRunning() bool {
	return t.running
}

func (t *Timer) IsOnce() bool {
	return t.once
}

func (t *Timer) ID() TimerID {
	return t.id
}

func (t *Timer) Duration() float64 {
	return t.duration
}

func (t *Timer) Elapsed() float64 {
	return t.elapsed
}

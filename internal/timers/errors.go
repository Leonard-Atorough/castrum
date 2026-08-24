package timers

import (
	"errors"
	"fmt"
)

var (
	ErrTimerNotFound        = errors.New("timer not found")
	ErrTimerInvalidDuration = errors.New("invalid timer duration")
	ErrTimerAlreadyRunning  = errors.New("timer already running")
	ErrTimerAlreadyStopped  = errors.New("timer already stopped")
)

type TimerError struct {
	TimerID TimerID
	Op      string
	Err     error
}

func (e *TimerError) Error() string {
	return fmt.Sprintf("timer %d: %s: %v", e.TimerID, e.Op, e.Err)
}

func (e *TimerError) Unwrap() error {
	return e.Err
}

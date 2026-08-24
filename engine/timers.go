package engine

import (
	"time"

	"github.com/leonard-atorough/castrum/internal/timers"
)

type TimersAPI struct {
	mgr *timers.TimerManager
}

func (t *TimersAPI) AddTimer(duration time.Duration, once bool, autoStart bool, callbackFunc func()) TimerID {
	return t.mgr.CreateTimer(duration.Seconds(), once, autoStart, callbackFunc)
}

func (t *TimersAPI) Start(timerID TimerID) error {
	return t.mgr.StartTimer(timerID)
}

func (t *TimersAPI) Pause(timerID TimerID) error {
	return t.mgr.StopTimer(timerID)
}

func (t *TimersAPI) Resume(timerID TimerID) error {
	return t.mgr.ResumeTimer(timerID)
}

func (t *TimersAPI) Cancel(timerID TimerID) error {
	return t.mgr.RemoveTimer(timerID)
}

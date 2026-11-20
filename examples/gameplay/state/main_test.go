package main

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/input"
)

func TestQueuedTransitionsHooksAndRestart(t *testing.T) {
	g, demo, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	advance := func(key input.Key) {
		t.Helper()
		snapshot := &input.Snapshot{}
		snapshot.Keys[key] = input.State{Pressed: true, Held: true}
		g.Context().Input = snapshot
		if err := g.Advance(time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	advance(input.KeySpace)
	if demo.machine.Current() != idle || !strings.Contains(demo.readout(), "Pending: playing") {
		t.Fatal("request was not displayed separately from the applied state")
	}
	if !reflect.DeepEqual(demo.events, []string{"enter idle", "initialized idle"}) {
		t.Fatalf("startup/request events=%v", demo.events)
	}
	advance(input.KeyNone)
	want := []string{"enter idle", "initialized idle", "exit idle", "enter playing", "observed idle -> playing"}
	if demo.machine.Current() != playing || !reflect.DeepEqual(demo.events, want) {
		t.Fatalf("transition events=%v", demo.events)
	}
	advance(input.KeyR)
	if pending, ok := demo.machine.Pending(); !ok || pending != playing {
		t.Fatal("restart was not queued")
	}
	advance(input.KeyNone)
	if demo.machine.Current() != playing || !reflect.DeepEqual(demo.events[3:], []string{
		"exit playing", "enter playing", "observed playing -> playing",
	}) {
		t.Fatalf("restart events=%v", demo.events)
	}
	advance(input.KeySpace)
	advance(input.KeyNone)
	if demo.machine.Current() != done {
		t.Fatal("playing did not transition to done")
	}
	advance(input.KeySpace)
	advance(input.KeyNone)
	if demo.machine.Current() != idle {
		t.Fatal("done did not return to idle")
	}
}

package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRelativeOrderProducesThePipelineResult(t *testing.T) {
	var out strings.Builder
	if err := run(&out); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "sample: 21\napply: 42\naudit: 42\nreport: 42\n"; got != want {
		t.Fatalf("pipeline=%q, want %q", got, want)
	}
}

var errOutput = errors.New("output unavailable")

type failingWriter struct{ calls int }

func (w *failingWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, errOutput
}

func TestOutputFailureStopsThePipeline(t *testing.T) {
	out := &failingWriter{}
	if err := run(out); !errors.Is(err, errOutput) || out.calls != 1 {
		t.Fatalf("error=%v writes=%d", err, out.calls)
	}
}

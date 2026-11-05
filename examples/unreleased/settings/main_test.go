package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Leonard-Atorough/castrum/settings"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRun(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out); err != nil {
		t.Fatal(err)
	}
	want := "Text size: 16, hints: [move]\nText size: 20, hints: [move jump]\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestFailedApplicationCanBeRetried(t *testing.T) {
	group, err := settings.New(preferences{TextSize: 16}, settings.Policy[preferences]{})
	if err != nil {
		t.Fatal(err)
	}
	if err := apply(failingWriter{}, group); err == nil {
		t.Fatal("failed application returned no error")
	}
	var out bytes.Buffer
	if err := apply(&out, group); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "Text size: 16, hints: []\n" {
		t.Fatalf("retry output = %q", got)
	}
	if err := apply(&out, group); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "Text size: 16, hints: []\nText size: 16, hints: []\n" {
		t.Fatalf("explicit application output = %q", got)
	}
}

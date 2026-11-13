package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/settings"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRun(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if err := run(root, &out); err != nil {
		t.Fatal(err)
	}
	want := "Text size: 16, hints: [move]\nText size: 20, hints: [move jump]\nSaved preferences to " + filepath.Join(root, "preferences.json") + "\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	saved, err := os.ReadFile(filepath.Join(root, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(root, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "Text size: 20, hints: [move jump]\nText size: 20, hints: [move jump]\n") {
		t.Fatalf("restored output = %q", out.String())
	}
	again, err := os.ReadFile(filepath.Join(root, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, saved) {
		t.Fatal("unchanged preferences did not save identically")
	}
}

func TestRunPreservesInvalidOrUnappliedPreferences(t *testing.T) {
	for _, tt := range []struct {
		name string
		data string
		fail bool
	}{
		{"malformed JSON", "{", false},
		{"invalid preferences", `{"text_size":0,"hints":["move"]}`, false},
		{"application failure", `{"text_size":24,"hints":["custom"]}`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "preferences.json")
			if err := os.WriteFile(path, []byte(tt.data), 0o600); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			var err error
			if tt.fail {
				err = run(root, failingWriter{})
			} else {
				err = run(root, &out)
			}
			if err == nil {
				t.Fatal("failed restore or application returned no error")
			}
			if out.Len() != 0 {
				t.Fatalf("invalid preferences were applied: %q", out.String())
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.data {
				t.Fatal("failed restore or application overwrote the saved document")
			}
		})
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

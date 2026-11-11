package core

import "testing"

func TestPhaseString(t *testing.T) {
	for _, tc := range []struct {
		phase Phase
		want  string
	}{
		{PhasePreStartup, "pre-startup"},
		{PhaseStartup, "startup"},
		{PhasePostStartup, "post-startup"},
		{PhasePreFrame, "pre-frame"},
		{PhaseFrame, "frame"},
		{PhasePostFrame, "post-frame"},
		{PhasePreFixed, "pre-fixed"},
		{PhaseFixed, "fixed"},
		{PhasePostFixed, "post-fixed"},
		{Phase(-1), "schedule(-1)"},
		{PhasePostFixed + 1, "schedule(9)"},
		{Phase(127), "schedule(127)"},
	} {
		if got := tc.phase.String(); got != tc.want {
			t.Errorf("Phase(%d).String() = %q, want %q", tc.phase, got, tc.want)
		}
	}
}

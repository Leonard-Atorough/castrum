package animation

import (
	"fmt"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
)

// tickTrace is one fixed tick's observable outcome for an Animator.
type tickTrace struct {
	State  string   // Current() after the tick
	Frames []string // "track:region:weight" per weighted layer, draw order
	Events []string // "Started:name", "Completed:name:loops"
}

// runScenario executes a test scenario for an Animator, recording tick traces.
func runScenario(t *testing.T, sys core.SystemFunc, sets Sets, anim Animator, dt time.Duration, ticks int, mutate func(tick int, a *Animator)) []tickTrace {
	t.Helper()
	w := core.NewWorld()
	if err := w.Provide(func(*core.World) (*signal.Hub, error) { return signal.NewHub(), nil }); err != nil {
		t.Fatal(err)
	}
	if err := w.InsertResource(sets); err != nil {
		t.Fatal(err)
	}
	entity, err := w.NewEntity(anim)
	if err != nil {
		t.Fatal(err)
	}
	hub := w.MustResource[*signal.Hub]()
	started := hub.NewReader[Started]()
	completed := hub.NewReader[Completed]()
	ctx := &core.Context{World: w, DeltaTime: dt}

	traces := make([]tickTrace, 0, ticks)
	for tick := 1; tick <= ticks; tick++ {
		if mutate != nil {
			if err := entity.Update(w, func(a *Animator) { mutate(tick, a) }); err != nil {
				t.Fatal(err)
			}
		}
		ctx.Tick = uint64(tick)
		if err := sys(ctx); err != nil {
			t.Fatalf("tick %d: %v", tick, err)
		}
		a, _ := entity.Component[Animator](w)
		tr := tickTrace{State: a.Current()}
		frames, err := a.Frames(sets)
		if err != nil {
			t.Fatalf("tick %d: Frames: %v", tick, err)
		}
		for _, f := range frames {
			tr.Frames = append(tr.Frames, fmt.Sprintf("%s:%s:%.2f", f.Track, f.Region, f.Weight))
		}
		for ev, ok := started.Next(); ok; ev, ok = started.Next() {
			tr.Events = append(tr.Events, "Started:"+ev.State)
		}
		for ev, ok := completed.Next(); ok; ev, ok = completed.Next() {
			tr.Events = append(tr.Events, fmt.Sprintf("Completed:%s:%d", ev.State, ev.Loops))
		}
		traces = append(traces, tr)
	}
	return traces
}

// testSets builds a small valid Sets: hero{idle(a,b loop, blended entry),
// attack(c once -> idle), die(d,e once), bounce(p,q,r ping-pong)}.
// EnterBlend lives on the transition target (idle), so attack -> idle blends.
func testSets(t *testing.T) Sets {
	t.Helper()
	track := func(frames ...string) []Track {
		return []Track{{Name: "body", Source: "atlas.json", Frames: frames, FPS: 10}}
	}
	sets, err := NewSets(map[string]Set{
		"hero": {
			Name: "hero",
			States: map[string]State{
				"idle":   {Name: "idle", Loop: Loop, EnterBlend: 0.25, Tracks: track("a", "b")},
				"attack": {Name: "attack", Next: "idle", Tracks: track("c")},
				"die":    {Name: "die", Tracks: track("d", "e")},
				"bounce": {Name: "bounce", Loop: LoopPingPong, Tracks: track("p", "q", "r")},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sets
}

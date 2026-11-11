package castrum

import (
	"reflect"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/settings"
)

func TestGameSettingsResourcesAndConsumers(t *testing.T) {
	type accessibility struct{ TextSize int }
	type gameplay struct{ Hints bool }

	game, err := New()
	if err != nil {
		t.Fatal(err)
	}
	prefs, err := settings.New(accessibility{TextSize: 16}, settings.Policy[accessibility]{})
	if err != nil {
		t.Fatal(err)
	}
	hints, err := settings.New(gameplay{Hints: true}, settings.Policy[gameplay]{})
	if err != nil {
		t.Fatal(err)
	}
	if err := game.World().Provide(func(*core.World) (*settings.Group[accessibility], error) {
		return prefs, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := game.World().Provide(func(*core.World) (*settings.Group[gameplay], error) {
		return hints, nil
	}); err != nil {
		t.Fatal(err)
	}
	resolved, err := game.World().Resource[*settings.Group[accessibility]]()
	if err != nil || resolved != prefs {
		t.Fatalf("accessibility resource: %p, error %v", resolved, err)
	}
	resolvedHints, err := game.World().Resource[*settings.Group[gameplay]]()
	if err != nil || resolvedHints != hints {
		t.Fatalf("gameplay resource: %p, error %v", resolvedHints, err)
	}

	var first, second []int
	for _, consumer := range []struct {
		name   string
		values *[]int
	}{
		{"settings.first", &first},
		{"settings.second", &second},
	} {
		if err := game.AddSystem(core.PhaseFrame, consumer.name, core.SystemFunc(func(*core.Context) error {
			value, err := resolved.Value()
			if err != nil {
				return err
			}
			*consumer.values = append(*consumer.values, value.TextSize)
			return nil
		})); err != nil {
			t.Fatal(err)
		}
	}
	if err := game.Startup(); err != nil {
		t.Fatal(err)
	}
	if err := game.Advance(time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := prefs.Update(func(value accessibility) accessibility {
		value.TextSize = 20
		return value
	}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := game.Advance(time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	for _, got := range [][]int{first, second} {
		if !reflect.DeepEqual(got, []int{16, 20, 20}) {
			t.Fatalf("consumer reads = %v, want [16 20 20]", got)
		}
	}
}

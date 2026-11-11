package audio

import (
	"math"
	"testing"
)

func TestAudioPreferencesValidation(t *testing.T) {
	prefs, err := NewPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := prefs.Value(); err != nil || got != (Preferences{Master: 1, Music: 1, SFX: 1}) {
		t.Fatalf("defaults = %+v, error %v", got, err)
	}
	valid := Preferences{Master: 0.5, Music: 0.25, SFX: 0}
	if err := prefs.Set(valid); err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		name string
		set  func(*Preferences, float64)
	}{
		{"master", func(p *Preferences, value float64) { p.Master = value }},
		{"music", func(p *Preferences, value float64) { p.Music = value }},
		{"sfx", func(p *Preferences, value float64) { p.SFX = value }},
	} {
		for _, invalid := range []struct {
			name  string
			value float64
		}{
			{"negative", -0.1},
			{"above unity", 1.1},
			{"nan", math.NaN()},
			{"positive infinity", math.Inf(1)},
			{"negative infinity", math.Inf(-1)},
		} {
			t.Run(field.name+"/"+invalid.name, func(t *testing.T) {
				if err := prefs.Update(func(p Preferences) Preferences {
					field.set(&p, invalid.value)
					return p
				}); err == nil {
					t.Fatal("invalid update succeeded")
				}
				got, err := prefs.Value()
				if err != nil || got != valid {
					t.Fatalf("after invalid update = %+v, error %v", got, err)
				}
			})
		}
	}
}

func TestPreferencesValidate(t *testing.T) {
	for _, valid := range []Preferences{
		{Master: 0, Music: 0, SFX: 0},
		{Master: 1, Music: 1, SFX: 1},
		{Master: 1, Music: 0.5, SFX: 0},
	} {
		if err := valid.Validate(); err != nil {
			t.Errorf("Validate(%+v): %v", valid, err)
		}
	}
}

func TestAudioPreferencesExplicitApplication(t *testing.T) {
	prefs, err := NewPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if err := prefs.Set(Preferences{Master: 0.8, Music: 0.6, SFX: 0.4}); err != nil {
		t.Fatal(err)
	}
	mixer := NewMixer()
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if err := mixer.SetMaster(value.Master); err != nil {
		t.Fatal(err)
	}
	if err := mixer.SetGroupVolume(GroupMusic, value.Music); err != nil {
		t.Fatal(err)
	}
	if err := mixer.SetGroupVolume(GroupSFX, value.SFX); err != nil {
		t.Fatal(err)
	}

	if err := mixer.SetMaster(0.3); err != nil {
		t.Fatal(err)
	}
	if err := prefs.Update(func(p Preferences) Preferences {
		p.Music = 0.2
		return p
	}); err != nil {
		t.Fatal(err)
	}
	if mixer.Master() != 0.3 || mixer.GroupVolume(GroupMusic) != 0.6 || mixer.GroupVolume(GroupSFX) != 0.4 {
		t.Fatal("preference update changed mixer state")
	}
	value, err = prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if err := mixer.SetGroupVolume(GroupMusic, value.Music); err != nil {
		t.Fatal(err)
	}
	if mixer.Master() != 0.3 || mixer.GroupVolume(GroupMusic) != 0.2 || mixer.GroupVolume(GroupSFX) != 0.4 {
		t.Fatal("explicit application changed the wrong mixer levels")
	}
}

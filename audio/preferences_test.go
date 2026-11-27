package audio

import (
	"math"
	"reflect"
	"testing"
)

func TestAudioPreferencesValidation(t *testing.T) {
	prefs, err := NewPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := prefs.Value(); err != nil || !reflect.DeepEqual(got, Preferences{Master: 1, Buses: map[string]float64{}}) {
		t.Fatalf("defaults = %+v, error %v", got, err)
	}
	valid := Preferences{Master: 0.5, Buses: map[string]float64{"music": 0.25, "sfx": 0}}
	if err := prefs.Set(valid); err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		name string
		set  func(*Preferences, float64)
	}{
		{"master", func(p *Preferences, value float64) { p.Master = value }},
		{"bus", func(p *Preferences, value float64) { p.Buses["music"] = value }},
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
				if err != nil || !reflect.DeepEqual(got, valid) {
					t.Fatalf("after invalid update = %+v, error %v", got, err)
				}
			})
		}
	}
}

func TestPreferencesValidate(t *testing.T) {
	for _, valid := range []Preferences{
		{Master: 0},
		{Master: 1, Buses: map[string]float64{"music": 1}},
		{Master: 1, Buses: map[string]float64{"music": 0.5, "sfx": 0}},
	} {
		if err := valid.Validate(); err != nil {
			t.Errorf("Validate(%+v): %v", valid, err)
		}
	}
}

func TestAudioPreferencesMapIsolation(t *testing.T) {
	prefs, err := NewPreferences()
	if err != nil {
		t.Fatal(err)
	}
	input := Preferences{Master: 1, Buses: map[string]float64{"music": 0.5}}
	if err := prefs.Set(input); err != nil {
		t.Fatal(err)
	}
	input.Buses["music"] = 0.1
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if value.Buses["music"] != 0.5 {
		t.Fatal("mutating input changed group")
	}
	value.Buses["music"] = 0.2
	var retained Preferences
	if err := prefs.Update(func(p Preferences) Preferences {
		if p.Buses["music"] != 0.5 {
			t.Fatal("mutating returned value changed group")
		}
		retained = p
		p.Buses["music"] = 0.7
		return p
	}); err != nil {
		t.Fatal(err)
	}
	retained.Buses["music"] = 0.3
	value, err = prefs.Value()
	if err != nil || value.Buses["music"] != 0.7 {
		t.Fatalf("group after mutating copies = %+v, error %v", value, err)
	}
}

func TestAudioPreferencesExplicitApplication(t *testing.T) {
	prefs, err := NewPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if err := prefs.Set(Preferences{Master: 0.8, Buses: map[string]float64{"music": 0.6, "sfx": 0.4}}); err != nil {
		t.Fatal(err)
	}
	mixer := NewMixer()
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if err := mixer.SetMasterVolume(value.Master); err != nil {
		t.Fatal(err)
	}
	for name, volume := range value.Buses {
		if err := mixer.SetBusVolume(BusName(name), volume); err != nil {
			t.Fatal(err)
		}
	}
	if err := prefs.Update(func(p Preferences) Preferences {
		p.Buses["music"] = 0.2
		return p
	}); err != nil {
		t.Fatal(err)
	}
	if mixer.MasterVolume() != 0.8 || mixer.BusVolume("music") != 0.6 || mixer.BusVolume("sfx") != 0.4 {
		t.Fatal("preference update changed mixer state")
	}
}

package audio

import (
	"math"
	"testing"
	"time"
)

func TestMixerResBusSetters(t *testing.T) {
	for _, tc := range []struct {
		name   string
		set    func(*testing.T, *Mixer)
		volume float64
		paused bool
		muted  bool
	}{
		{"volume", func(t *testing.T, m *Mixer) {
			if err := m.SetBusVolume("music", 0.4); err != nil {
				t.Fatal(err)
			}
		}, 0.4, false, false},
		{"paused", func(_ *testing.T, m *Mixer) { m.SetBusPaused("music", true) }, 1, true, false},
		{"muted", func(_ *testing.T, m *Mixer) { m.SetBusMuted("music", true) }, 1, false, true},
	} {
		for _, makeMixer := range []struct {
			name string
			new  func() *Mixer
		}{
			{"constructed", NewMixer},
			{"zero value", func() *Mixer { return &Mixer{} }},
		} {
			t.Run(tc.name+"/"+makeMixer.name, func(t *testing.T) {
				m := makeMixer.new()
				tc.set(t, m)
				if !m.HasBus("music") || m.BusVolume("music") != tc.volume ||
					m.BusPaused("music") != tc.paused || m.BusMuted("music") != tc.muted {
					t.Fatal("bus creation did not preserve the requested value and defaults")
				}

				if err := m.SetBusVolume("music", 0.7); err != nil {
					t.Fatal(err)
				}
				m.SetBusPaused("music", true)
				m.SetBusMuted("music", true)
				if m.BusVolume("music") != 0.7 || !m.BusPaused("music") || !m.BusMuted("music") {
					t.Fatal("setting a bus field changed another field")
				}
			})
		}
	}
}

func TestMixerResBusVolumeRejectsNaN(t *testing.T) {
	m := NewMixer()
	if err := m.SetBusVolume("new", math.NaN()); err == nil || m.HasBus("new") {
		t.Fatal("NaN should be rejected without creating a bus")
	}
	if err := m.SetBusVolume("existing", 0.5); err != nil {
		t.Fatal(err)
	}
	if err := m.SetBusVolume("existing", math.NaN()); err == nil || m.BusVolume("existing") != 0.5 {
		t.Fatal("NaN should be rejected without changing the existing bus")
	}
}

func TestMixerResGlobalAndBusPause(t *testing.T) {
	m := NewMixer()
	m.SetBusPaused("music", true)
	m.SetBusPaused("sfx", false)

	m.PauseAll()
	if !m.BusPaused("music") || !m.BusPaused("sfx") || !m.BusPaused("unconfigured") {
		t.Fatal("global pause should affect configured and unconfigured buses")
	}

	m.ResumeAll()
	if !m.BusPaused("music") || m.BusPaused("sfx") || m.BusPaused("unconfigured") {
		t.Fatal("resuming globally should preserve individual bus pauses")
	}
}

func TestMixerResFadeVolumes(t *testing.T) {
	m := NewMixer()
	if err := m.SetMasterVolume(0.8); err != nil {
		t.Fatal(err)
	}
	if err := m.SetBusVolume("music", 0.6); err != nil {
		t.Fatal(err)
	}
	if err := m.FadeMaster(0.2, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.Fade("music", 0.2, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.Fade("new", 0, time.Second); err != nil {
		t.Fatal(err)
	}
	m.PauseAll()
	if err := m.update(500 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.MasterVolume()-0.5) > 1e-9 || math.Abs(m.BusVolume("music")-0.4) > 1e-9 || m.BusVolume("new") != 0.5 {
		t.Fatalf("halfway volumes: master=%v music=%v new=%v", m.MasterVolume(), m.BusVolume("music"), m.BusVolume("new"))
	}
	if err := m.update(time.Second); err != nil {
		t.Fatal(err)
	}
	if m.MasterVolume() != 0.2 || m.BusVolume("music") != 0.2 || m.BusVolume("new") != 0 {
		t.Fatalf("final volumes: master=%v music=%v new=%v", m.MasterVolume(), m.BusVolume("music"), m.BusVolume("new"))
	}
}

func TestMixerResFadeInterruption(t *testing.T) {
	m := NewMixer()
	if err := m.FadeMaster(0, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.Fade("music", 0, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.update(500 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := m.FadeMaster(1, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.Fade("music", 1, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.update(500 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if m.MasterVolume() != 0.75 || m.BusVolume("music") != 0.75 {
		t.Fatal("replacement fades did not start at the current volumes")
	}
	if err := m.SetMasterVolume(0.3); err != nil {
		t.Fatal(err)
	}
	if err := m.SetBusVolume("music", 0.4); err != nil {
		t.Fatal(err)
	}
	if err := m.update(time.Second); err != nil {
		t.Fatal(err)
	}
	if m.MasterVolume() != 0.3 || m.BusVolume("music") != 0.4 {
		t.Fatal("explicit volume changes did not cancel fades")
	}
}

func TestMixerResFadeValidation(t *testing.T) {
	m := NewMixer()
	for _, tc := range []struct {
		name     string
		target   float64
		duration time.Duration
	}{
		{"nan", math.NaN(), time.Second},
		{"zero duration", 0, 0},
		{"negative duration", 0, -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := m.Fade("new", tc.target, tc.duration); err == nil || m.HasBus("new") {
				t.Fatal("invalid fade should not create a bus")
			}
			if err := m.FadeMaster(tc.target, tc.duration); err == nil {
				t.Fatal("invalid master fade succeeded")
			}
		})
	}
	if err := m.FadeMaster(0, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.Fade("music", 0, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.FadeMaster(math.NaN(), time.Second); err == nil {
		t.Fatal("invalid replacement fade succeeded")
	}
	if err := m.Fade("music", 1, 0); err == nil {
		t.Fatal("invalid replacement bus fade succeeded")
	}
	if err := m.SetMasterVolume(math.NaN()); err == nil {
		t.Fatal("invalid master volume succeeded")
	}
	if err := m.SetBusVolume("music", math.NaN()); err == nil {
		t.Fatal("invalid bus volume succeeded")
	}
	if err := m.update(-time.Second); err == nil || m.MasterVolume() != 1 {
		t.Fatal("negative delta should fail without changing volume")
	}
	if err := m.update(500 * time.Millisecond); err != nil || m.MasterVolume() != 0.5 || m.BusVolume("music") != 0.5 {
		t.Fatalf("invalid operations changed active fades: master=%v music=%v error=%v", m.MasterVolume(), m.BusVolume("music"), err)
	}
	if err := m.Fade("clamped", 2, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.update(time.Second); err != nil || m.BusVolume("clamped") != 1 {
		t.Fatalf("fade target was not clamped: volume=%v error=%v", m.BusVolume("clamped"), err)
	}
}

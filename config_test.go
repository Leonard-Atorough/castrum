package castrum

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateConfig_SetsSensibleDefaults(t *testing.T) {
	cfg := &Config{}

	ValidateConfig(cfg)

	if cfg.Project.Name != "My Project" {
		t.Fatalf("Project.Name = %q, want %q", cfg.Project.Name, "My Project")
	}
	if cfg.Window.Width != 800 {
		t.Fatalf("Window.Width = %d, want 800", cfg.Window.Width)
	}
	if cfg.Window.Height != 600 {
		t.Fatalf("Window.Height = %d, want 600", cfg.Window.Height)
	}
	if cfg.Window.Mode != "windowed" {
		t.Fatalf("Window.Mode = %q, want %q", cfg.Window.Mode, "windowed")
	}
	if cfg.Graphics.VirtualWidth != 800 {
		t.Fatalf("Graphics.VirtualWidth = %d, want 800", cfg.Graphics.VirtualWidth)
	}
	if cfg.Graphics.VirtualHeight != 600 {
		t.Fatalf("Graphics.VirtualHeight = %d, want 600", cfg.Graphics.VirtualHeight)
	}
	if cfg.Graphics.ClearColor != "#000000" {
		t.Fatalf("Graphics.ClearColor = %q, want %q", cfg.Graphics.ClearColor, "#000000")
	}
	if cfg.Engine.TicksPerSecond != 60 {
		t.Fatalf("Engine.TicksPerSecond = %d, want 60", cfg.Engine.TicksPerSecond)
	}
	if cfg.Engine.MaxFPS != 60 {
		t.Fatalf("Engine.MaxFPS = %d, want 60", cfg.Engine.MaxFPS)
	}
}

func TestValidateConfig_ClampsAndNormalizes(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: ""},
		Window: WindowConfig{
			Width:     0,
			Height:    -1,
			Mode:      "invalid",
			ScaleMode: "bad",
		},
		Graphics: GraphicsConfig{
			VirtualWidth:  0,
			VirtualHeight: -20,
			ClearColor:    "bad",
			Filtering:     "invalid",
		},
		Audio: AudioConfig{
			Enabled:      true,
			MasterVolume: -2,
			MusicVolume:  2,
			SFXVolume:    0.5,
		},
		Input: InputConfig{
			MouseVisible:   false,
			GamepadEnabled: false,
		},
		Engine: EngineConfig{
			TicksPerSecond: 0,
			MaxFPS:         -10,
		},
	}

	ValidateConfig(cfg)

	if cfg.Project.Name != "My Project" {
		t.Fatalf("Project.Name = %q, want %q", cfg.Project.Name, "My Project")
	}
	if cfg.Window.Width != 800 {
		t.Fatalf("Window.Width = %d, want 800", cfg.Window.Width)
	}
	if cfg.Window.Height != 600 {
		t.Fatalf("Window.Height = %d, want 600", cfg.Window.Height)
	}
	if cfg.Window.Mode != "windowed" {
		t.Fatalf("Window.Mode = %q, want %q", cfg.Window.Mode, "windowed")
	}
	if cfg.Window.ScaleMode != "stretch" {
		t.Fatalf("Window.ScaleMode = %q, want %q", cfg.Window.ScaleMode, "stretch")
	}
	if cfg.Graphics.VirtualWidth != 800 {
		t.Fatalf("Graphics.VirtualWidth = %d, want 800", cfg.Graphics.VirtualWidth)
	}
	if cfg.Graphics.VirtualHeight != 600 {
		t.Fatalf("Graphics.VirtualHeight = %d, want 600", cfg.Graphics.VirtualHeight)
	}
	if cfg.Graphics.ClearColor != "#000000" {
		t.Fatalf("Graphics.ClearColor = %q, want %q", cfg.Graphics.ClearColor, "#000000")
	}
	if cfg.Graphics.Filtering != "linear" {
		t.Fatalf("Graphics.Filtering = %q, want %q", cfg.Graphics.Filtering, "linear")
	}
	if cfg.Audio.MasterVolume != 0 {
		t.Fatalf("Audio.MasterVolume = %v, want 0", cfg.Audio.MasterVolume)
	}
	if cfg.Audio.MusicVolume != 1 {
		t.Fatalf("Audio.MusicVolume = %v, want 1", cfg.Audio.MusicVolume)
	}
	if cfg.Audio.SFXVolume != 0.5 {
		t.Fatalf("Audio.SFXVolume = %v, want 0.5", cfg.Audio.SFXVolume)
	}
	if cfg.Engine.TicksPerSecond != 60 {
		t.Fatalf("Engine.TicksPerSecond = %d, want 60", cfg.Engine.TicksPerSecond)
	}
	if cfg.Engine.MaxFPS != 60 {
		t.Fatalf("Engine.MaxFPS = %d, want 60", cfg.Engine.MaxFPS)
	}
}

func TestValidateConfig_AllowsNilPointer(t *testing.T) {
	var cfg *Config
	ValidateConfig(cfg)
}

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid YAML config", func(t *testing.T) {
		yaml := `project:
  name: "Test Game"
  version: "2.0.0"
window:
  width: 1024
  height: 768
  title: "Test"
graphics:
  virtual_width: 512
  virtual_height: 384
engine:
  ticks_per_second: 30`

		config, err := LoadConfig(strings.NewReader(yaml))
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if config.Project.Name != "Test Game" {
			t.Fatalf("Project.Name = %q, want %q", config.Project.Name, "Test Game")
		}
		if config.Window.Width != 1024 {
			t.Fatalf("Window.Width = %d, want 1024", config.Window.Width)
		}
		if config.Engine.TicksPerSecond != 30 {
			t.Fatalf("Engine.TicksPerSecond = %d, want 30", config.Engine.TicksPerSecond)
		}
	})

	t.Run("returns error on invalid YAML", func(t *testing.T) {
		yaml := `project:
  name: Test Game
  invalid: [broken yaml`

		_, err := LoadConfig(strings.NewReader(yaml))
		if err == nil {
			t.Fatal("LoadConfig should return error for invalid YAML")
		}
	})

	t.Run("validates loaded config", func(t *testing.T) {
		yaml := `project:
  name: ""`

		config, err := LoadConfig(strings.NewReader(yaml))
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		// Should have default values applied
		if config.Project.Name != "My Project" {
			t.Fatalf("Expected default project name, got %q", config.Project.Name)
		}
	})
}

func TestSaveConfig(t *testing.T) {
	config := DefaultConfig()
	config.Project.Name = "Save Test"
	config.Window.Width = 1920

	buf := &bytes.Buffer{}
	err := config.SaveConfig(buf)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Save Test") {
		t.Fatalf("Saved config doesn't contain project name: %s", output)
	}
	if !strings.Contains(output, "1920") {
		t.Fatalf("Saved config doesn't contain window width: %s", output)
	}
}

func TestNormalizeWindowMode(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"fullscreen", "fullscreen"},
		{"FULLSCREEN", "fullscreen"},
		{"full", "fullscreen"},
		{"exclusive", "fullscreen"},
		{"  fullscreen  ", "fullscreen"},
		{"borderless", "borderless"},
		{"BORDERLESS", "borderless"},
		{"borderless_fullscreen", "borderless"},
		{"windowed", "windowed"},
		{"WINDOWED", "windowed"},
		{"window", "windowed"},
		{"", "windowed"},
		{"invalid", "windowed"},
		{"  ", "windowed"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeWindowMode(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeWindowMode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeScaleMode(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"fit", "fit"},
		{"FIT", "fit"},
		{"fill", "fill"},
		{"FILL", "fill"},
		{"stretch", "stretch"},
		{"STRETCH", "stretch"},
		{"", "stretch"},
		{"invalid", "stretch"},
		{"  ", "stretch"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeScaleMode(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeScaleMode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeFiltering(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"nearest", "nearest"},
		{"NEAREST", "nearest"},
		{"pixel", "nearest"},
		{"point", "nearest"},
		{"linear", "linear"},
		{"LINEAR", "linear"},
		{"smooth", "linear"},
		{"bilinear", "linear"},
		{"", "linear"},
		{"invalid", "linear"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeFiltering(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeFiltering(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeHexColor(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"#000000", "#000000"},
		{"#FFFFFF", "#FFFFFF"},
		{"#fff", "#fff"},
		{"#FFF", "#FFF"},
		{"#FF0000AA", "#FF0000AA"},
		{"", "#000000"},
		{"   ", "#000000"},
		{"000000", "#000000"},
		{"invalid", "#000000"},
		{"#GGGGGG", "#000000"},
		{"#12345", "#000000"},
		{"#12345678", "#12345678"},
		{"#00", "#000000"},
		{" #123456 ", "#123456"},
		{"#AbCdEf", "#AbCdEf"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeHexColor(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeHexColor(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

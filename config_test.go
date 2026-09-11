package castrum

import "testing"

func TestValidateConfig_SetsSensibleDefaults(t *testing.T) {
	cfg := &Config{}

	ValidateConfig(cfg)

	if cfg.Project.Name != "My Project" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "My Project")
	}
	if cfg.Window.Width != 800 {
		t.Errorf("Window.Width = %d, want 800", cfg.Window.Width)
	}
	if cfg.Window.Height != 600 {
		t.Errorf("Window.Height = %d, want 600", cfg.Window.Height)
	}
	if cfg.Window.Mode != "windowed" {
		t.Errorf("Window.Mode = %q, want %q", cfg.Window.Mode, "windowed")
	}
	if cfg.Graphics.VirtualWidth != 800 {
		t.Errorf("Graphics.VirtualWidth = %d, want 800", cfg.Graphics.VirtualWidth)
	}
	if cfg.Graphics.VirtualHeight != 600 {
		t.Errorf("Graphics.VirtualHeight = %d, want 600", cfg.Graphics.VirtualHeight)
	}
	if cfg.Graphics.ClearColor != "#000000" {
		t.Errorf("Graphics.ClearColor = %q, want %q", cfg.Graphics.ClearColor, "#000000")
	}
	if cfg.Engine.TicksPerSecond != 60 {
		t.Errorf("Engine.TicksPerSecond = %d, want 60", cfg.Engine.TicksPerSecond)
	}
	if cfg.Engine.MaxFPS != 60 {
		t.Errorf("Engine.MaxFPS = %d, want 60", cfg.Engine.MaxFPS)
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
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "My Project")
	}
	if cfg.Window.Width != 800 {
		t.Errorf("Window.Width = %d, want 800", cfg.Window.Width)
	}
	if cfg.Window.Height != 600 {
		t.Errorf("Window.Height = %d, want 600", cfg.Window.Height)
	}
	if cfg.Window.Mode != "windowed" {
		t.Errorf("Window.Mode = %q, want %q", cfg.Window.Mode, "windowed")
	}
	if cfg.Window.ScaleMode != "stretch" {
		t.Errorf("Window.ScaleMode = %q, want %q", cfg.Window.ScaleMode, "stretch")
	}
	if cfg.Graphics.VirtualWidth != 800 {
		t.Errorf("Graphics.VirtualWidth = %d, want 800", cfg.Graphics.VirtualWidth)
	}
	if cfg.Graphics.VirtualHeight != 600 {
		t.Errorf("Graphics.VirtualHeight = %d, want 600", cfg.Graphics.VirtualHeight)
	}
	if cfg.Graphics.ClearColor != "#000000" {
		t.Errorf("Graphics.ClearColor = %q, want %q", cfg.Graphics.ClearColor, "#000000")
	}
	if cfg.Graphics.Filtering != "linear" {
		t.Errorf("Graphics.Filtering = %q, want %q", cfg.Graphics.Filtering, "linear")
	}
	if cfg.Audio.MasterVolume != 0 {
		t.Errorf("Audio.MasterVolume = %v, want 0", cfg.Audio.MasterVolume)
	}
	if cfg.Audio.MusicVolume != 1 {
		t.Errorf("Audio.MusicVolume = %v, want 1", cfg.Audio.MusicVolume)
	}
	if cfg.Audio.SFXVolume != 0.5 {
		t.Errorf("Audio.SFXVolume = %v, want 0.5", cfg.Audio.SFXVolume)
	}
	if cfg.Engine.TicksPerSecond != 60 {
		t.Errorf("Engine.TicksPerSecond = %d, want 60", cfg.Engine.TicksPerSecond)
	}
	if cfg.Engine.MaxFPS != 60 {
		t.Errorf("Engine.MaxFPS = %d, want 60", cfg.Engine.MaxFPS)
	}
}

func TestValidateConfig_AllowsNilPointer(t *testing.T) {
	var cfg *Config
	ValidateConfig(cfg)
}

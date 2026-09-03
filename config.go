package castrum

import (
	"io"
	"strings"

	"github.com/leonard-atorough/castrum/geom"
	"go.yaml.in/yaml/v3"
)

type Orientation string

const (
	OrientationLandscape Orientation = "landscape"
	OrientationPortrait  Orientation = "portrait"
)

type Config struct {
	Project  ProjectConfig   `yaml:"project"`
	Window   WindowConfig    `yaml:"window"`
	Graphics GraphicsConfig  `yaml:"graphics"`
	Audio    AudioConfig     `yaml:"audio"`
	Input    InputConfig     `yaml:"input"`
	Engine   EngineConfig    `yaml:"engine"`
	World    CollisionConfig `yaml:"world"`
}

type ProjectConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	IconPath    string `yaml:"icon_path"`
}

type WindowConfig struct {
	Title      string `yaml:"title"`
	Width      int    `yaml:"width"`
	Height     int    `yaml:"height"`
	Mode       string `yaml:"mode"`
	Resizable  bool   `yaml:"resizable"`
	Borderless bool   `yaml:"borderless"`
	Fullscreen bool   `yaml:"fullscreen"`
	VSync      bool   `yaml:"vsync"`
	ScaleMode  string `yaml:"scale_mode"`
}

type GraphicsConfig struct {
	VirtualWidth  int    `yaml:"virtual_width"`
	VirtualHeight int    `yaml:"virtual_height"`
	PixelPerfect  bool   `yaml:"pixel_perfect"`
	ClearColor    string `yaml:"clear_color"`
	Filtering     string `yaml:"filtering"`
}

type AudioConfig struct {
	Enabled      bool    `yaml:"enabled"`
	MasterVolume float64 `yaml:"master_volume"`
	MusicVolume  float64 `yaml:"music_volume"`
	SFXVolume    float64 `yaml:"sfx_volume"`
}

type InputConfig struct {
	MouseVisible   bool `yaml:"mouse_visible"`
	GamepadEnabled bool `yaml:"gamepad_enabled"`
	// Need to add more input configurations as needed
	// Keyboard config and keybinds could be handled using a typed map
}

type EngineConfig struct {
	TicksPerSecond   int  `yaml:"ticks_per_second"`
	MaxFPS           int  `yaml:"max_fps"`
	FixedDeltaTime   bool `yaml:"fixed_delta_time"`
	PauseOnFocusLost bool `yaml:"pause_on_focus_lost"`
	EnableDebug      bool `yaml:"enable_debug"`
	EnableLogging    bool `yaml:"enable_logging"`
}

type CollisionConfig struct {
	BoundingArea    geom.Rect `yaml:"bounds"`
	GridCellSize    float64   `yaml:"grid_cell_size"`
	EnableDebugDraw bool      `yaml:"enable_debug_draw"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	ValidateConfig(&config)
	return &config, nil
}

func ValidateConfig(config *Config) {
	if config == nil {
		return
	}

	if strings.TrimSpace(config.Project.Name) == "" {
		config.Project.Name = "My Project"
	}
	if strings.TrimSpace(config.Project.Description) == "" {
		config.Project.Description = "A description of my project"
	}
	if strings.TrimSpace(config.Project.Version) == "" {
		config.Project.Version = "1.0.0"
	}
	if strings.TrimSpace(config.Project.Author) == "" {
		config.Project.Author = "Author Name"
	}
	if strings.TrimSpace(config.Project.IconPath) == "" {
		config.Project.IconPath = "path/to/icon.png"
	}

	if config.Window.Width <= 0 {
		config.Window.Width = 800
	}
	if config.Window.Height <= 0 {
		config.Window.Height = 600
	}
	if strings.TrimSpace(config.Window.Title) == "" {
		config.Window.Title = config.Project.Name
	}
	config.Window.Mode = normalizeWindowMode(config.Window.Mode)
	config.Window.ScaleMode = normalizeScaleMode(config.Window.ScaleMode)

	if config.Graphics.VirtualWidth <= 0 {
		config.Graphics.VirtualWidth = config.Window.Width
	}
	if config.Graphics.VirtualHeight <= 0 {
		config.Graphics.VirtualHeight = config.Window.Height
	}
	config.Graphics.ClearColor = normalizeHexColor(config.Graphics.ClearColor)
	config.Graphics.Filtering = normalizeFiltering(config.Graphics.Filtering)

	if config.Audio.MasterVolume < 0 {
		config.Audio.MasterVolume = 0
	}
	if config.Audio.MasterVolume > 1 {
		config.Audio.MasterVolume = 1
	}
	if config.Audio.MusicVolume < 0 {
		config.Audio.MusicVolume = 0
	}
	if config.Audio.MusicVolume > 1 {
		config.Audio.MusicVolume = 1
	}
	if config.Audio.SFXVolume < 0 {
		config.Audio.SFXVolume = 0
	}
	if config.Audio.SFXVolume > 1 {
		config.Audio.SFXVolume = 1
	}

	if config.Engine.TicksPerSecond <= 0 {
		config.Engine.TicksPerSecond = 60
	}
	if config.Engine.MaxFPS <= 0 {
		config.Engine.MaxFPS = 60
	}
	if config.World.GridCellSize <= 0 {
		config.World.GridCellSize = 256
	}
}

func DefaultConfig() *Config {
	cfg := &Config{
		Project: ProjectConfig{
			Name:        "My Project",
			Description: "A description of my project",
			Version:     "1.0.0",
			Author:      "Author Name",
			IconPath:    "path/to/icon.png",
		},
		Window: WindowConfig{
			Title:      "My Project",
			Width:      800,
			Height:     600,
			Mode:       "windowed",
			Resizable:  true,
			Borderless: false,
			Fullscreen: false,
			VSync:      true,
			ScaleMode:  "stretch",
		},
		Graphics: GraphicsConfig{
			VirtualWidth:  800,
			VirtualHeight: 600,
			PixelPerfect:  false,
			ClearColor:    "#000000",
			Filtering:     "linear",
		},
		Audio: AudioConfig{
			Enabled:      true,
			MasterVolume: 1.0,
			MusicVolume:  1.0,
			SFXVolume:    1.0,
		},
		Input: InputConfig{
			MouseVisible:   true,
			GamepadEnabled: true,
		},
		Engine: EngineConfig{
			TicksPerSecond:   60,
			MaxFPS:           60,
			FixedDeltaTime:   true,
			PauseOnFocusLost: true,
			EnableDebug:      false,
			EnableLogging:    true,
		},
		World: CollisionConfig{
			BoundingArea:    geom.Rect{}, //default to unbounded
			GridCellSize:    256,         // Used for spatial partitioning of the world grid
			EnableDebugDraw: false,
		},
	}
	ValidateConfig(cfg)
	return cfg
}

func normalizeWindowMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "fullscreen", "full", "exclusive":
		return "fullscreen"
	case "borderless", "borderless_fullscreen":
		return "borderless"
	case "windowed", "window", "":
		return "windowed"
	default:
		return "windowed"
	}
}

func normalizeScaleMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "fit":
		return "fit"
	case "fill":
		return "fill"
	case "stretch", "":
		return "stretch"
	default:
		return "stretch"
	}
}

func normalizeFiltering(filtering string) string {
	switch strings.ToLower(strings.TrimSpace(filtering)) {
	case "nearest", "pixel", "point":
		return "nearest"
	case "linear", "smooth", "bilinear", "":
		return "linear"
	default:
		return "linear"
	}
}

func normalizeHexColor(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "#000000"
	}
	if !strings.HasPrefix(trimmed, "#") {
		return "#000000"
	}
	if len(trimmed) != 4 && len(trimmed) != 7 && len(trimmed) != 9 {
		return "#000000"
	}
	for _, ch := range trimmed[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", ch) {
			return "#000000"
		}
	}
	return trimmed
}

func (c *Config) SaveConfig(writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()

	return encoder.Encode(c)
}

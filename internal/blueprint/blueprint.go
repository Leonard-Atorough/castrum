package blueprint

import (
	"io"

	"go.yaml.in/yaml/v3"
)

type Blueprint struct {
	Name       string      `yaml:"name"`
	Components []Component `yaml:"components"`
	Version    string      `yaml:"version"`
}

type Component struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

func (b *Blueprint) Construct(reader io.Reader) error {
	decoder := yaml.NewDecoder(reader)
	return decoder.Decode(b)
}


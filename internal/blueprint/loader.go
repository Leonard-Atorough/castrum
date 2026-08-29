package blueprint

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/leonard-atorough/castrum/internal/core"
	"go.yaml.in/yaml/v3"
)

type Loader struct {
	reg *Registry
}

func NewLoader(reg *Registry) *Loader {
	return &Loader{reg: reg}
}

func (l *Loader) Load(reader io.Reader) (*Blueprint, error) {
	var blueprint Blueprint
	decoder := yaml.NewDecoder(reader)
	err := decoder.Decode(&blueprint)
	if err != nil {
		return nil, err
	}
	return &blueprint, nil
}

func (l *Loader) LoadFromPath(path string) (*Blueprint, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return l.Load(file)
}

func (l *Loader) LoadFromString(data string) (*Blueprint, error) {
	return l.Load(strings.NewReader(data))
}

func (l *Loader) LoadFromBytes(data []byte) (*Blueprint, error) {
	return l.Load(bytes.NewReader(data))
}

func (l *Loader) Spawn(world *core.World, bp *Blueprint) (*core.Entity, error) {
	return bp.Construct(world, l.reg)
}

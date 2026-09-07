package assets

import (
	"fmt"
	"io/fs"
	"os"

	"go.yaml.in/yaml/v3"
)

// AnimationClip defines a reusable animation: immutable, shared across entities.
type AnimationClip struct {
	Frames    []string `yaml:"frames"`
	FrameSpeed float64 `yaml:"frameSpeed"`
	Loop      bool     `yaml:"loop"`
}

type animationStore struct {
	fs        fs.FS
	Animations map[string]*AnimationClip
}

func newAnimationStore(filesystem fs.FS) *animationStore {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &animationStore{
		fs:        filesystem,
		Animations: make(map[string]*AnimationClip),
	}
}

func (s *animationStore) Load(path string) (*AnimationClip, error) {
	// Check cache first
	if clip, ok := s.Animations[path]; ok {
		return clip, nil
	}

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var clip AnimationClip
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&clip); err != nil {
		return nil, fmt.Errorf("failed to decode animation clip: %w", err)
	}

	// Validate clip
	if len(clip.Frames) == 0 {
		return nil, fmt.Errorf("animation clip has no frames")
	}
	if clip.FrameSpeed <= 0 {
		return nil, fmt.Errorf("animation frame speed must be positive")
	}

	s.Animations[path] = &clip // cache by path
	return &clip, nil
}

// Package builtin provides engine-shipped assets, currently the Go Regular
// font. Games use [Font] to identify it; [castrum.New] makes the built-in path
// available alongside game assets.
package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/Leonard-Atorough/castrum/asset"
)

//go:embed fonts/GoRegular.ttf
var files embed.FS

// Font is the asset path for the built-in Go Regular font.
const Font = "builtin/fonts/GoRegular.ttf"

// WithFilesystem returns a filesystem that serves built-in assets and resolves
// all other paths from filesystem. Use it with [asset.NewLoader] when not using
// [castrum.New]. If filesystem is nil, other paths resolve from the current
// working directory. Paths under "builtin/" are reserved for engine-shipped
// assets.
func WithFilesystem(filesystem fs.FS) fs.FS {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return filesystemWithBuiltins{base: filesystem}
}

// Register loads and caches the built-in font. Loaders created by
// [castrum.New] can resolve it automatically; standalone loaders must use
// [WithFilesystem].
func Register(loader *asset.Loader) error {
	if loader == nil {
		return fmt.Errorf("builtin: loader cannot be nil")
	}
	if _, err := loader.Load[asset.FontData](Font); err != nil {
		return fmt.Errorf("builtin: register font: %w", err)
	}
	return nil
}

type filesystemWithBuiltins struct {
	base fs.FS
}

func (f filesystemWithBuiltins) Open(name string) (fs.File, error) {
	const prefix = "builtin/"
	if after, ok := strings.CutPrefix(name, prefix); ok {
		return files.Open(after)
	}
	return f.base.Open(name)
}

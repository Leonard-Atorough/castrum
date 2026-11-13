package asset

import (
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	_ fs.FS     = (*Directory)(nil)
	_ SaveStore = (*Directory)(nil)
)

// Directory reads files and publishes encoded documents beneath an open root.
// It implements [fs.FS] and [SaveStore]. The caller must close it after all
// loaders, savers, and open files have finished using it.
type Directory struct {
	root *os.Root
}

// NewDirectory opens an existing directory at path. It does not create the
// directory or select an application-data location.
func NewDirectory(path string) (*Directory, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &Directory{root: root}, nil
}

// Open opens a slash-separated name relative to the directory.
// The caller owns the returned file and must close it.
func (d *Directory) Open(name string) (fs.File, error) {
	return d.root.FS().Open(name)
}

// Write encodes a document to a private staging file and replaces name after
// encoding and closing succeed. Missing parent directories are created.
// Name must satisfy [fs.ValidPath], must not be ".", and must not contain
// backslashes.
//
// Encoding or close failures leave the destination untouched. Failed writes
// attempt staging cleanup and preserve cleanup errors alongside the cause.
// Replacement uses the platform's rename operation; atomicity and crash
// durability are not guaranteed. Callers must order writes to the same name.
func (d *Directory) Write(name string, encode func(io.Writer) error) (err error) {
	if name == "." || !fs.ValidPath(name) || strings.Contains(name, "\\") {
		return &fs.PathError{Op: "write", Path: name, Err: fs.ErrInvalid}
	}
	if encode == nil {
		return &fs.PathError{Op: "write", Path: name, Err: errors.New("encode callback cannot be nil")}
	}
	destination, err := filepath.Localize(name)
	if err != nil {
		return &fs.PathError{Op: "write", Path: name, Err: err}
	}
	parent := filepath.Dir(destination)
	if err := d.root.MkdirAll(parent, 0o700); err != nil {
		return err
	}

	stage, stageName, err := d.createStagingFile(parent)
	if err != nil {
		return err
	}
	closed, published := false, false
	defer func() {
		if !closed {
			err = errors.Join(err, stage.Close())
		}
		if !published {
			removeErr := d.root.Remove(stageName)
			if !errors.Is(removeErr, fs.ErrNotExist) {
				err = errors.Join(err, removeErr)
			}
		}
	}()

	if err = encode(stage); err != nil {
		return err
	}

	closed = true
	if err = stage.Close(); err != nil {
		return err
	}

	if err = d.root.Rename(stageName, destination); err != nil {
		return err
	}
	published = true
	return nil
}

// Close releases the directory's root handle.
func (d *Directory) Close() error {
	return d.root.Close()
}

func (d *Directory) createStagingFile(parent string) (*os.File, string, error) {
	for range 100 {
		name := filepath.Join(parent, ".castrum-"+rand.Text()+".tmp")
		file, err := d.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return file, name, err
	}
	return nil, "", &fs.PathError{Op: "create staging file", Path: parent, Err: fs.ErrExist}
}

package asset

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
)

func newTestDirectory(t *testing.T) *Directory {
	t.Helper()
	directory, err := NewDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	})
	return directory
}

func directoryFiles(t *testing.T, directory *Directory) []string {
	t.Helper()
	var names []string
	if err := fs.WalkDir(directory, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			names = append(names, name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return names
}

func TestDirectoryRoundTrip(t *testing.T) {
	directory := newTestDirectory(t)
	codecs := NewCodecs()
	if err := codecs.SetJSON[spriteMeta](); err != nil {
		t.Fatal(err)
	}
	saver, err := NewSaver(directory, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, directory, LoaderOptions{Codecs: codecs, CachePolicy: CacheBypass})
	for _, name := range []string{"profile.json", "profiles/player.json"} {
		for _, value := range []spriteMeta{{Name: "old"}, {Name: "new"}} {
			if err := saver.Save(name, value); err != nil {
				t.Fatal(err)
			}
			got, err := loader.Load[spriteMeta](name)
			if err != nil || got != value {
				t.Fatalf("round trip %q = %+v, %v; want %+v", name, got, err, value)
			}
		}
	}
	if got := directoryFiles(t, directory); !slices.Equal(got, []string{"profile.json", "profiles/player.json"}) {
		t.Fatalf("unexpected files after publication: %v", got)
	}
}

func TestDirectoryStagingLocationAndPermissions(t *testing.T) {
	directory := newTestDirectory(t)
	calls := 0
	if err := directory.Write("profiles/player.json", func(w io.Writer) error {
		calls++
		entries, err := fs.ReadDir(directory, "profiles")
		if err != nil {
			return err
		}
		if len(entries) != 1 {
			return fmt.Errorf("destination directory contains %d staging files, want 1", len(entries))
		}
		info, err := entries[0].Info()
		if err != nil {
			return err
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("staging permissions = %o, want no group or other access", info.Mode().Perm())
		}
		_, err = io.WriteString(w, "new")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("encode called %d times, want 1", calls)
	}
}

func TestDirectoryRejectsPathsBeforeEncoding(t *testing.T) {
	directory := newTestDirectory(t)
	for _, name := range []string{
		"", ".", "..", "../outside.json", "/outside.json",
		"profiles/../player.json", `profiles\player.json`,
		filepath.Join(t.TempDir(), "outside.json"),
	} {
		t.Run(name, func(t *testing.T) {
			called := false
			err := directory.Write(name, func(io.Writer) error {
				called = true
				return nil
			})
			if err == nil || called {
				t.Fatalf("Write(%q) = %v; encode called = %t", name, err, called)
			}
		})
	}
	if got := directoryFiles(t, directory); len(got) != 0 {
		t.Fatalf("invalid names created files: %v", got)
	}
}

func TestDirectoryRejectsNilEncoder(t *testing.T) {
	directory := newTestDirectory(t)
	if err := directory.Write("profiles/player.json", nil); err == nil {
		t.Fatal("nil encoder accepted")
	}
	entries, err := fs.ReadDir(directory, ".")
	if err != nil || len(entries) != 0 {
		t.Fatalf("nil encoder changed directory: %v, %v", entries, err)
	}
}

func TestDirectoryFailuresPreserveDestination(t *testing.T) {
	for _, failure := range []string{"encode", "write", "close", "publish", "panic"} {
		t.Run(failure, func(t *testing.T) {
			directory := newTestDirectory(t)
			const name = "profiles/player.json"
			const old = "old document"
			if err := directory.Write(name, func(w io.Writer) error {
				_, err := io.WriteString(w, old)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			target := name
			if failure == "publish" {
				target = "profiles"
			}
			encodeErr := errors.New("encode failed")
			calls := 0
			var err error
			var panicValue any
			func() {
				defer func() { panicValue = recover() }()
				err = directory.Write(target, func(w io.Writer) error {
					calls++
					if failure == "write" || failure == "close" {
						// Force descriptor failures without a mock filesystem.
						if err := w.(*os.File).Close(); err != nil {
							return err
						}
						if failure == "close" {
							return nil
						}
					}
					if _, err := io.WriteString(w, "partial"); err != nil {
						return err
					}
					switch failure {
					case "encode":
						return encodeErr
					case "panic":
						panic(encodeErr)
					default:
						return nil
					}
				})
			}()
			if calls != 1 {
				t.Fatalf("encode called %d times, want 1", calls)
			}
			if failure == "panic" {
				if panicValue != encodeErr {
					t.Fatalf("panic = %v, want %v", panicValue, encodeErr)
				}
			} else if err == nil || panicValue != nil {
				t.Fatalf("failed write = %v; panic = %v", err, panicValue)
			}
			if failure == "encode" && !errors.Is(err, encodeErr) {
				t.Fatalf("lost encoding error: %v", err)
			}
			if (failure == "write" || failure == "close") && !errors.Is(err, os.ErrClosed) {
				t.Fatalf("lost descriptor error: %v", err)
			}
			content, readErr := fs.ReadFile(directory, name)
			if readErr != nil || string(content) != old {
				t.Fatalf("failed write changed destination: %q, %v", content, readErr)
			}
			if got := directoryFiles(t, directory); !slices.Equal(got, []string{name}) {
				t.Fatalf("failed write leaked staging files: %v", got)
			}
		})
	}
}

func TestDirectoryConcurrentStaging(t *testing.T) {
	directory := newTestDirectory(t)
	const peers = 8
	var arrived, done sync.WaitGroup
	arrived.Add(peers)
	release := make(chan struct{})
	errs := make([]error, peers)
	for i := range peers {
		done.Go(func() {
			called := false
			errs[i] = directory.Write(fmt.Sprintf("profiles/%d.txt", i), func(w io.Writer) error {
				called = true
				arrived.Done()
				<-release
				_, err := fmt.Fprint(w, i)
				return err
			})
			if !called {
				arrived.Done()
			}
		})
	}
	arrived.Wait()
	entries, err := fs.ReadDir(directory, "profiles")
	close(release)
	done.Wait()
	if err != nil || len(entries) != peers {
		t.Fatalf("concurrent staging = %d entries, %v; want %d", len(entries), err, peers)
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v", i, err)
		}
		content, err := fs.ReadFile(directory, fmt.Sprintf("profiles/%d.txt", i))
		if err != nil || string(content) != fmt.Sprint(i) {
			t.Fatalf("writer %d published %q, %v", i, content, err)
		}
	}
	if got := directoryFiles(t, directory); len(got) != peers {
		t.Fatalf("concurrent writes leaked staging files: %v", got)
	}
}

func TestDirectoryCleanupFailurePreservesCause(t *testing.T) {
	directory := newTestDirectory(t)
	encodeErr := errors.New("encode failed")
	var cleanupCause error
	err := directory.Write("profiles/player.json", func(w io.Writer) error {
		if err := w.(*os.File).Close(); err != nil {
			return err
		}
		matches, err := fs.Glob(directory, "profiles/.castrum-*.tmp")
		if err != nil || len(matches) != 1 {
			return fmt.Errorf("find staging file: %v, %v", matches, err)
		}
		stageName, err := filepath.Localize(matches[0])
		if err != nil {
			return err
		}
		if err := directory.root.Remove(stageName); err != nil {
			return err
		}
		// A nonempty directory at the staging name makes cleanup fail portably.
		if err := directory.root.Mkdir(stageName, 0o700); err != nil {
			return err
		}
		if err := directory.root.WriteFile(filepath.Join(stageName, "blocker"), nil, 0o600); err != nil {
			return err
		}
		var pathErr *fs.PathError
		if err := directory.root.Remove(stageName); !errors.As(err, &pathErr) {
			return fmt.Errorf("expected nonempty directory removal to fail: %v", err)
		}
		cleanupCause = pathErr.Err
		return encodeErr
	})
	if !errors.Is(err, encodeErr) || cleanupCause == nil || !errors.Is(err, cleanupCause) {
		t.Fatalf("cleanup lost a cause: %v; want %v and %v", err, encodeErr, cleanupCause)
	}
}

func TestDirectoryCannotFollowSymlinkOutsideRoot(t *testing.T) {
	directory := newTestDirectory(t)
	outside := t.TempDir()
	target := filepath.Join(outside, "profile.json")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory.root.Name(), "link")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if file, err := directory.Open("link/profile.json"); err == nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("read followed a symlink outside the root")
	}
	called := false
	err := directory.Write("link/profile.json", func(w io.Writer) error {
		called = true
		_, err := io.WriteString(w, "new")
		return err
	})
	if err == nil || called {
		t.Fatalf("escaping write = %v; encode called = %t", err, called)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "old" {
		t.Fatalf("outside document changed: %q, %v", content, err)
	}
}

func TestDirectoryLifetime(t *testing.T) {
	path := t.TempDir()
	if _, err := NewDirectory(filepath.Join(path, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing root = %v, want fs.ErrNotExist", err)
	}
	filePath := filepath.Join(path, "file.txt")
	if err := os.WriteFile(filePath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if directory, err := NewDirectory(filePath); err == nil {
		directory.Close()
		t.Fatal("regular file accepted as a directory")
	}
	directory, err := NewDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := directory.Open("."); err == nil {
		t.Fatal("closed directory accepted a read")
	}
	called := false
	if err := directory.Write("profile.json", func(io.Writer) error {
		called = true
		return nil
	}); err == nil || called {
		t.Fatalf("closed directory write = %v; encode called = %t", err, called)
	}
}

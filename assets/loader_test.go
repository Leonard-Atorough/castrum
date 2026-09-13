package assets

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

type loadTestAsset struct {
	Value string
}

func newLoadTestAssets(t *testing.T, contents string) *Assets {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "asset.test"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	assets := NewAssets(os.DirFS(directory))
	if err := assets.loader.RegisterDecoder(Format("test"), func(_ context.Context, reader io.Reader) (loadTestAsset, error) {
		data, err := io.ReadAll(reader)
		if err != nil {
			return loadTestAsset{}, err
		}
		return loadTestAsset{Value: string(data)}, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}
	return assets
}

func TestLoaderCachesDecodedAssets(t *testing.T) {
	assets := newLoadTestAssets(t, "first")
	ctx := context.Background()

	first, err := assets.loader.Load[loadTestAsset](ctx, "asset.test")
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}
	if first.Value != "first" {
		t.Fatalf("first value = %q, want %q", first.Value, "first")
	}

	assets.loader.Invalidate(ID("asset.test"))
	if _, ok := assets.loader.service.Cached("asset.test", reflect.TypeFor[loadTestAsset](), "test"); ok {
		t.Fatal("Invalidate left the decoded value cached")
	}
}

func TestLoaderNormalizesFSPathBeforeOpening(t *testing.T) {
	assets := NewAssets(fstest.MapFS{
		"asset.test": &fstest.MapFile{Data: []byte("normalized")},
	})
	if err := assets.loader.RegisterDecoder(Format("test"), func(_ context.Context, reader io.Reader) (loadTestAsset, error) {
		data, err := io.ReadAll(reader)
		if err != nil {
			return loadTestAsset{}, err
		}
		return loadTestAsset{Value: string(data)}, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}

	value, err := assets.loader.Load[loadTestAsset](context.Background(), "nested/../asset.test")
	if err != nil {
		t.Fatalf("Load with non-canonical path failed: %v", err)
	}
	if value.Value != "normalized" {
		t.Fatalf("value = %q, want %q", value.Value, "normalized")
	}
}

func TestSavePathUsesCanonicalInvalidationID(t *testing.T) {
	directory := t.TempDir()
	path := directory + string(os.PathSeparator) + "nested" + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "asset.test"
	cleanPath := normalizeSavePath(path)
	if cleanPath == path {
		t.Fatal("test path was already canonical")
	}
	if got := assetIDForSavePath(path); got != ID(filepath.ToSlash(cleanPath)) {
		t.Fatalf("save ID = %q, want %q", got, filepath.ToSlash(cleanPath))
	}
}

func TestLoaderCachePolicyNoneBypassesCache(t *testing.T) {
	assets := newLoadTestAssets(t, "value")
	ctx := context.Background()

	if _, err := assets.loader.Load[loadTestAsset](ctx, "asset.test"); err != nil {
		t.Fatalf("cached Load failed: %v", err)
	}
	if _, err := assets.loader.Load[loadTestAsset](ctx, "asset.test", WithCache(CachePolicyNone)); err != nil {
		t.Fatalf("uncached Load failed: %v", err)
	}
}

func TestLoaderMissingFileReturnsAssetError(t *testing.T) {
	assets := newLoadTestAssets(t, "value")

	_, err := assets.loader.Load[loadTestAsset](context.Background(), "missing.test")
	if err == nil {
		t.Fatal("Load succeeded for a missing file")
	}
	var assetErr *AssetError
	if !errors.As(err, &assetErr) || assetErr.Source != "Load" {
		t.Fatalf("error = %T %v, want AssetError from Load", err, err)
	}
}

func TestLoaderMissingDecoderReturnsAssetError(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "asset.unknown"), []byte("value"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	loader := NewLoader(os.DirFS(directory))

	_, err := loader.Load[loadTestAsset](context.Background(), "asset.unknown")
	if err == nil {
		t.Fatal("Load succeeded without a registered decoder")
	}
	var assetErr *AssetError
	if !errors.As(err, &assetErr) || assetErr.Source != "Load" {
		t.Fatalf("error = %T %v, want AssetError from Load", err, err)
	}
}

func TestLoaderDecoderErrorIsWrapped(t *testing.T) {
	assets := newLoadTestAssets(t, "value")
	sentinel := errors.New("decoder failed")
	if err := assets.loader.RegisterDecoder(Format("test-error"), func(context.Context, io.Reader) (loadTestAsset, error) {
		return loadTestAsset{}, sentinel
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}

	_, err := assets.loader.Load[loadTestAsset](context.Background(), "asset.test", WithFormat(Format("test-error")))
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want wrapped decoder error", err)
	}
}

func TestLoaderSingleflightDecodesOnce(t *testing.T) {
	assets := newLoadTestAssets(t, "value")
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	if err := assets.loader.RegisterDecoder(Format("singleflight"), func(context.Context, io.Reader) (loadTestAsset, error) {
		calls.Add(1)
		close(started)
		<-release
		return loadTestAsset{Value: "decoded"}, nil
	}, false); err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}

	results := make(chan loadTestAsset, 2)
	errorsCh := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := assets.loader.Load[loadTestAsset](context.Background(), "asset.test", WithFormat(Format("singleflight")))
			results <- value
			errorsCh <- err
		}()
	}
	<-started
	close(release)
	group.Wait()
	close(results)
	close(errorsCh)

	for err := range errorsCh {
		if err != nil {
			t.Fatalf("coalesced Load failed: %v", err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("decoder calls = %d, want 1", got)
	}
}

func TestLoaderReaderWithoutDecoderReturnsAssetError(t *testing.T) {
	loader := NewLoader(nil)

	_, err := loader.LoadReader[loadTestAsset](context.Background(), strings.NewReader("value"), WithFormat(Format("missing")))
	if err == nil {
		t.Fatal("LoadReader succeeded without a decoder")
	}
	var assetErr *AssetError
	if !errors.As(err, &assetErr) || assetErr.Source != "LoadReader" {
		t.Fatalf("error = %T %v, want AssetError from LoadReader", err, err)
	}
}

func TestSaverInvalidationNotifiesListeners(t *testing.T) {
	assets := newLoadTestAssets(t, "old")
	var notified ID
	assets.loader.RegisterInvalidationListener(func(id ID) {
		notified = id
	})

	if err := assets.saver.RegisterEncoder(Format("test"), func(_ context.Context, writer io.Writer, value loadTestAsset) error {
		_, err := io.WriteString(writer, value.Value)
		return err
	}, false); err != nil {
		t.Fatalf("RegisterEncoder failed: %v", err)
	}

	path := filepath.Join(t.TempDir(), "asset.test")
	if err := assets.saver.SavePath(context.Background(), path, loadTestAsset{Value: "new"}); err != nil {
		t.Fatalf("SavePath failed: %v", err)
	}
	if notified != ID(filepath.ToSlash(filepath.Clean(path))) {
		t.Fatalf("notified ID = %q, want %q", notified, filepath.ToSlash(filepath.Clean(path)))
	}
}

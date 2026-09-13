package assets

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type saveTestAsset struct {
	Value string
}

func newSaveTestAssets(t *testing.T) *Assets {
	t.Helper()
	assets := NewAssets(nil)
	if err := assets.Saver.RegisterEncoder[saveTestAsset](FormatYAML, func(_ context.Context, writer io.Writer, value saveTestAsset) error {
		_, err := fmt.Fprint(writer, value.Value)
		return err
	}, false); err != nil {
		t.Fatalf("RegisterEncoder failed: %v", err)
	}
	return assets
}

func TestSaverSaveWriter(t *testing.T) {
	assets := newSaveTestAssets(t)
	var output bytes.Buffer

	err := assets.Saver.Save(context.Background(), &output, saveTestAsset{Value: "writer"}, WithSaveFormat(FormatYAML))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if output.String() != "writer" {
		t.Fatalf("Save output = %q, want %q", output.String(), "writer")
	}
}

func TestSaverSavePathInfersFormatAndCreatesDirectory(t *testing.T) {
	assets := newSaveTestAssets(t)
	directory := t.TempDir()
	assetPath := filepath.Join(directory, "nested", "asset.yaml")

	err := assets.Saver.SavePath(context.Background(), assetPath, saveTestAsset{Value: "first"})
	if err != nil {
		t.Fatalf("SavePath failed: %v", err)
	}

	data, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("saved data = %q, want %q", string(data), "first")
	}

	err = assets.Saver.SavePath(context.Background(), assetPath, saveTestAsset{Value: "second"})
	if err != nil {
		t.Fatalf("replacement SavePath failed: %v", err)
	}
	data, err = os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("ReadFile after replacement failed: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("replaced data = %q, want %q", string(data), "second")
	}
}

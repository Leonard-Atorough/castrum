package assets

import (
	"context"
	"testing"
	"testing/fstest"
	"time"
)

func TestNewAssets(t *testing.T) {
	a := NewAssets(nil)
	if a.Textures == nil || a.Blueprints == nil {
		t.Fatal("expected both stores to be initialized")
	}
}

func TestAssetsLoad(t *testing.T) {
	fs := fstest.MapFS{
		"test.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	tests := []struct {
		name      string
		path      string
		wantNil   bool
		wantError bool
	}{
		{"blueprint .yaml", "test.yaml", false, false},
		{"unknown extension", "test.txt", true, true},
		{"no extension", "noext", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := a.LoadSync(tt.path)
			if (err != nil) != tt.wantError {
				t.Fatalf("Load(%q): got err %v, wantError %v", tt.path, err, tt.wantError)
			}
			if (res == nil) != tt.wantNil {
				t.Fatalf("Load(%q): got nil %v, wantNil %v", tt.path, res == nil, tt.wantNil)
			}
		})
	}
}

func TestLoadAsync(t *testing.T) {
	fs := fstest.MapFS{
		"test.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	tests := []struct {
		name      string
		path      string
		wantNil   bool
		wantError bool
	}{
		{"blueprint .yaml", "test.yaml", false, false},
		{"unknown extension", "test.txt", true, true},
		{"no extension", "noext", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result := <-a.LoadAsync(ctx, tt.path)
			if (result.Err != nil) != tt.wantError {
				t.Fatalf("LoadAsync(%q): got err %v, wantError %v", tt.path, result.Err, tt.wantError)
			}
			if (result.Value == nil) != tt.wantNil {
				t.Fatalf("LoadAsync(%q): got nil %v, wantNil %v", tt.path, result.Value == nil, tt.wantNil)
			}
		})
	}
}

func TestLoadAsyncCancellation(t *testing.T) {
	fs := fstest.MapFS{
		"test.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := <-a.LoadAsync(ctx, "test.yaml")
	if result.Err == nil {
		t.Fatal("LoadAsync: expected cancellation error, got nil")
	}
	if result.Err != context.Canceled {
		t.Fatalf("LoadAsync: expected context.Canceled, got %v", result.Err)
	}
}

func TestLoadAsyncTimeout(t *testing.T) {
	fs := fstest.MapFS{
		"test.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Give the timeout time to fire
	time.Sleep(10 * time.Millisecond)

	result := <-a.LoadAsync(ctx, "test.yaml")
	if result.Err == nil {
		t.Fatal("LoadAsync: expected timeout error, got nil")
	}
	if result.Err != context.DeadlineExceeded {
		t.Fatalf("LoadAsync: expected context.DeadlineExceeded, got %v", result.Err)
	}
}

func TestLoadBatch(t *testing.T) {
	fs := fstest.MapFS{
		"test1.yaml": {Data: []byte(validBlueprintYAML)},
		"test2.yaml": {Data: []byte(validBlueprintYAML)},
		"test3.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := a.LoadBatch(ctx, []string{"test1.yaml", "test2.yaml", "test3.yaml"})
	if err != nil {
		t.Fatalf("LoadBatch: got unexpected error %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("LoadBatch: expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if result.Err != nil {
			t.Errorf("LoadBatch[%d]: got error %v", i, result.Err)
		}
		if result.Value == nil {
			t.Errorf("LoadBatch[%d]: got nil value", i)
		}
	}
}

func TestLoadBatchPartialFailure(t *testing.T) {
	fs := fstest.MapFS{
		"test1.yaml": {Data: []byte(validBlueprintYAML)},
		"test2.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Mix valid and invalid paths
	results, err := a.LoadBatch(ctx, []string{"test1.yaml", "missing.yaml", "test2.yaml"})
	if err != nil {
		t.Fatalf("LoadBatch: got unexpected error %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("LoadBatch: expected 3 results, got %d", len(results))
	}

	// test1.yaml should succeed
	if results[0].Err != nil {
		t.Errorf("LoadBatch[0]: expected success, got error %v", results[0].Err)
	}

	// missing.yaml should fail
	if results[1].Err == nil {
		t.Errorf("LoadBatch[1]: expected error for missing file, got nil")
	}

	// test2.yaml should succeed
	if results[2].Err != nil {
		t.Errorf("LoadBatch[2]: expected success, got error %v", results[2].Err)
	}
}

func TestLoadBatchCancellation(t *testing.T) {
	fs := fstest.MapFS{
		"test1.yaml": {Data: []byte(validBlueprintYAML)},
		"test2.yaml": {Data: []byte(validBlueprintYAML)},
		"test3.yaml": {Data: []byte(validBlueprintYAML)},
	}

	a := NewAssets(fs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before submitting

	_, err := a.LoadBatch(ctx, []string{"test1.yaml", "test2.yaml", "test3.yaml"})
	if err != context.Canceled {
		t.Fatalf("LoadBatch: expected context.Canceled, got %v", err)
	}
}

package asset

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func newSheetFixture(t *testing.T) *Sheet {
	t.Helper()
	sheet, err := NewSheet("./textures/../tex.png", map[string]SheetRegion{
		"b": {X: 2, W: 2, H: 2}, "a": {W: 2, H: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func TestSheetCopiesRegionsAndUsesOrdinaryJSON(t *testing.T) {
	regions := map[string]SheetRegion{"b": {X: 2, W: 2, H: 2}, "a": {W: 2, H: 2}}
	sheet, err := NewSheet("./textures/../tex.png", regions)
	if err != nil {
		t.Fatal(err)
	}
	regions["b"] = SheetRegion{X: 99, W: 2, H: 2}
	got, err := sheet.Region("b")
	if err != nil || got != image.Rect(2, 0, 4, 2) {
		t.Fatalf("input mutation affected sheet: %v, %v", got, err)
	}
	data, err := json.Marshal(sheet)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"texture":"tex.png","regions":{"a":{"x":0,"y":0,"w":2,"h":2},"b":{"x":2,"y":0,"w":2,"h":2}}}`
	if string(data) != want {
		t.Fatalf("JSON = %s, want %s", data, want)
	}
	loader := newTestLoader(t, nil, LoaderOptions{})
	loaded, err := loader.LoadReader[Sheet](bytes.NewReader(data), FormatJSON)
	if err != nil {
		t.Fatal(err)
	}
	if err := loaded.ValidateBounds(image.Rect(0, 0, 4, 2)); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(loaded)
	if err != nil || !bytes.Equal(data, encoded) {
		t.Fatalf("default JSON codec changed document: %s, %v", encoded, err)
	}
	if _, err := loaded.Region("missing"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing region = %v", err)
	}
	if err := loaded.ValidateBounds(image.Rect(0, 0, 3, 2)); err == nil {
		t.Fatal("accepted image bounds smaller than a region")
	}
}

func TestSheetValidation(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	good := map[string]SheetRegion{"a": {W: 1, H: 1}}
	for _, tc := range []struct {
		name    string
		texture ID
		regions map[string]SheetRegion
	}{
		{"empty texture", "", good},
		{"escaping texture", "../tex.png", good},
		{"absolute texture", "/tex.png", good},
		{"native texture", `assets\tex.png`, good},
		{"hidden native segment", `assets\native/../tex.png`, good},
		{"hidden drive segment", "C:/../tex.png", good},
		{"URL texture", "https://example.com/tex.png", good},
		{"empty regions", "tex.png", nil},
		{"empty name", "tex.png", map[string]SheetRegion{"": {W: 1, H: 1}}},
		{"invalid UTF-8 name", "tex.png", map[string]SheetRegion{"\xff": {W: 1, H: 1}}},
		{"negative x", "tex.png", map[string]SheetRegion{"a": {X: -1, W: 1, H: 1}}},
		{"negative y", "tex.png", map[string]SheetRegion{"a": {Y: -1, W: 1, H: 1}}},
		{"zero width", "tex.png", map[string]SheetRegion{"a": {H: 1}}},
		{"negative height", "tex.png", map[string]SheetRegion{"a": {W: 1, H: -1}}},
		{"overflow x", "tex.png", map[string]SheetRegion{"a": {X: maxInt, W: 1, H: 1}}},
		{"overflow y", "tex.png", map[string]SheetRegion{"a": {Y: maxInt, W: 1, H: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := NewSheet(tc.texture, tc.regions); err == nil || got != nil {
				t.Fatalf("NewSheet = %v, %v", got, err)
			}
			if err := (Sheet{Texture: tc.texture, Regions: tc.regions}).Validate(); err == nil {
				t.Fatal("invalid public document passed validation")
			}
		})
	}
	loader := newTestLoader(t, nil, LoaderOptions{})
	for _, data := range []string{"null", "{}", `{"texture":"tex.png","regions":{"a":{"w":0,"h":1}}}`} {
		sheet, err := loader.LoadReader[Sheet](strings.NewReader(data), FormatJSON)
		var assetErr *AssetError
		if !errors.As(err, &assetErr) || assetErr.Op != "Decode" || sheet.Texture != "" || sheet.Regions != nil {
			t.Fatalf("invalid metadata %s = %+v, %v; want zero value and Decode error", data, sheet, err)
		}
	}
	sheet := Sheet{Texture: "tex.png", Regions: map[string]SheetRegion{"bad": {W: 0, H: 1}}}
	if _, err := sheet.Region("bad"); err == nil {
		t.Fatal("selected invalid geometry was returned")
	}
	codecs := NewCodecs()
	if err := codecs.SetJSON[Sheet](); err != nil {
		t.Fatal(err)
	}
	overridden := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	if _, err := overridden.LoadReader[Sheet](strings.NewReader("null"), FormatJSON); err != nil {
		t.Fatalf("explicit SetJSON replacement should retain ordinary JSON decoding: %v", err)
	}
}

func TestSheetLoadRejectsInvalidMetadataBeforeCaching(t *testing.T) {
	for name, data := range map[string]string{
		"null":           "null",
		"empty document": "{}",
		"empty regions":  `{"texture":"tex.png","regions":{}}`,
		"escaping path":  `{"texture":"../tex.png","regions":{"a":{"w":1,"h":1}}}`,
		"invalid unused": `{"texture":"tex.png","regions":{"a":{"w":1,"h":1},"unused":{"w":0,"h":1}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			file := &fstest.MapFile{Data: []byte(data)}
			files := &sheetCountingFS{FS: fstest.MapFS{"sheet.json": file}}
			loader := newTestLoader(t, files, LoaderOptions{})
			got, err := loader.Load[Sheet]("./sheet.json")
			var assetErr *AssetError
			if !errors.As(err, &assetErr) || assetErr.Op != "Decode" || assetErr.ID != "sheet.json" ||
				got.Texture != "" || got.Regions != nil {
				t.Fatalf("invalid load = %+v, %v; want zero value and path-aware Decode error", got, err)
			}
			file.Data = []byte(`{"texture":"tex.png","regions":{"a":{"w":1,"h":1}}}`)
			for range 2 {
				got, err = loader.Load[Sheet]("sheet.json")
				if err != nil || got.Texture != "tex.png" || len(got.Regions) != 1 {
					t.Fatalf("corrected load = %+v, %v", got, err)
				}
			}
			if files.opens != 2 {
				t.Fatalf("opened %d files; want failed read, retry, then cache reuse with no texture read", files.opens)
			}
		})
	}
}

func TestSheetSaveAndFileResolution(t *testing.T) {
	directory, err := NewDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	})
	saver, err := NewSaver(directory, SaverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	sheet := newSheetFixture(t)
	if err := sheet.Save(saver, "sheets/hero.sheet"); err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, directory, LoaderOptions{})
	got, err := FileSheet("sheets/hero.sheet").Resolve(loader)
	if err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(sheet)
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(got)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("save/load changed sheet: %s / %s, %v", first, second, err)
	}
	if _, err := directory.Open("tex.png"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Save unexpectedly copied the texture: %v", err)
	}
	if err := sheet.Save(nil, "hero.json"); err == nil {
		t.Fatal("nil saver accepted")
	}
	if err := (*Sheet)(nil).Save(saver, "hero.json"); err == nil {
		t.Fatal("nil sheet saved")
	}
	if err := (&Sheet{}).Save(saver, "hero.json"); err == nil {
		t.Fatal("invalid sheet saved")
	}
	unconfigured, err := NewSaver(directory, SaverOptions{Codecs: &Codecs{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := sheet.Save(unconfigured, "hero.json"); err == nil {
		t.Fatal("saving without a sheet encoder succeeded")
	}
}

type sheetCountingFS struct {
	fs.FS
	opens int
}

func (f *sheetCountingFS) Open(name string) (fs.File, error) {
	f.opens++
	return f.FS.Open(name)
}

func TestSheetReferencesResolveMemoryOrCachedFiles(t *testing.T) {
	sheet := newSheetFixture(t)
	data, err := json.Marshal(sheet)
	if err != nil {
		t.Fatal(err)
	}
	files := &sheetCountingFS{FS: fstest.MapFS{
		"sheets/hero.json":  {Data: data},
		"sheets/null.json":  {Data: []byte("null")},
		"sheets/extra.json": {Data: append(slices.Clone(data), []byte(" {}")...)},
		"sheets/alias.json": {Data: []byte(`{"texture":"./tex.png","regions":{"a":{"w":1,"h":1}}}`)},
	}}
	loader := newTestLoader(t, files, LoaderOptions{})
	ref := FileSheet("./sheets/hero.json")
	if ref != FileSheet("sheets/hero.json") {
		t.Fatal("equivalent file paths produced different references")
	}
	for range 2 {
		got, err := ref.Resolve(loader)
		if err != nil || got.Texture != "tex.png" {
			t.Fatalf("file resolve = %v, %v", got, err)
		}
	}
	if files.opens != 1 {
		t.Fatalf("file reference opened %d times, want one", files.opens)
	}
	if got, err := MemorySheet(sheet).Resolve(nil); err != nil || got.Texture != "tex.png" || files.opens != 1 {
		t.Fatalf("memory resolve = %v, %v; opens=%d", got, err, files.opens)
	}
	bypass := newTestLoader(t, files, LoaderOptions{CachePolicy: CacheBypass})
	for range 2 {
		if _, err := ref.Resolve(bypass); err != nil {
			t.Fatal(err)
		}
	}
	if files.opens != 3 {
		t.Fatalf("bypass references opened %d times, want three total reads", files.opens)
	}
	if _, err := ref.Resolve(loader); err != nil || files.opens != 3 {
		t.Fatalf("bypass disturbed original cached reference: %v; opens=%d", err, files.opens)
	}
	for _, ref := range []SheetRef{SheetRef{}, MemorySheet(nil), MemorySheet(&Sheet{}), FileSheet("../hero.json"),
		FileSheet(`sheets\hero.json`), FileSheet(`sheets\native/../hero.json`), FileSheet("C:/../hero.json")} {
		if err := ref.Validate(); err == nil {
			t.Fatalf("invalid reference accepted: %+v", ref)
		}
		if _, err := ref.Resolve(loader); err == nil {
			t.Fatalf("invalid reference resolved: %+v", ref)
		}
	}
	if _, err := FileSheet("sheets/hero.json").Resolve(nil); err == nil {
		t.Fatal("file reference resolved without loader")
	}
	for _, name := range []ID{"missing.json", "sheets/null.json", "sheets/extra.json"} {
		if _, err := FileSheet(name).Resolve(loader); err == nil || !strings.Contains(err.Error(), string(name)) {
			t.Fatalf("bad file %q resolved: %v", name, err)
		}
	}
	alias, err := FileSheet("sheets/alias.json").Resolve(loader)
	if err != nil || alias.Texture != "tex.png" {
		t.Fatalf("document texture was not normalized: %q, %v", alias.Texture, err)
	}
	unconfigured := newTestLoader(t, files, LoaderOptions{Codecs: &Codecs{}})
	if _, err := ref.Resolve(unconfigured); err == nil || !strings.Contains(err.Error(), "decoder") {
		t.Fatalf("reference without a decoder = %v", err)
	}
}

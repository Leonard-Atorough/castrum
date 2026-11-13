package spritesheet

import (
	"image"
	"testing"
)

func TestGridSlicingAndBounds(t *testing.T) {
	bounds := image.Rect(2, 4, 6, 8)
	sheet, err := Grid("tex.png", bounds, 2, 2, "tile")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]image.Rectangle{
		"tile_0": image.Rect(2, 4, 4, 6), "tile_1": image.Rect(4, 4, 6, 6),
		"tile_2": image.Rect(2, 6, 4, 8), "tile_3": image.Rect(4, 6, 6, 8),
	} {
		got, err := sheet.Region(name)
		if err != nil || got != want {
			t.Fatalf("%s = %v, %v; want %v", name, got, err, want)
		}
	}
	if err := sheet.ValidateBounds(bounds); err != nil {
		t.Fatal(err)
	}
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		bounds image.Rectangle
		w, h   int
		prefix string
	}{
		{image.Rectangle{}, 2, 2, "tile"},
		{image.Rect(-1, 0, 4, 4), 2, 2, "tile"},
		{bounds, 0, 2, "tile"},
		{bounds, 2, -1, "tile"},
		{bounds, 3, 2, "tile"},
		{bounds, 2, 3, "tile"},
		{bounds, 2, 2, ""},
		{image.Rect(0, 0, maxInt, maxInt), 1, 1, "tile"},
	} {
		if got, err := Grid("tex.png", tc.bounds, tc.w, tc.h, tc.prefix); err == nil || got != nil {
			t.Fatalf("invalid grid %+v = %v, %v", tc, got, err)
		}
	}
}

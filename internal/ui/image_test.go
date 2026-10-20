package ui

import (
	"image"
	"testing"
)

func TestFit(t *testing.T) {
	for name, tc := range map[string]struct {
		w, h, maxCols, maxRows int
		cols, rows             int
	}{
		"limited by width":  {1000, 200, 50, 16, 50, 5},
		"limited by height": {480, 384, 50, 16, 40, 16},
		"narrow terminal":   {480, 384, 30, 16, 30, 12},
		"tall image":        {100, 1000, 50, 16, 3, 16},
	} {
		t.Run(name, func(t *testing.T) {
			cols, rows := fit(image.Rect(0, 0, tc.w, tc.h), tc.maxCols, tc.maxRows)
			if cols != tc.cols || rows != tc.rows {
				t.Errorf("fit = %dx%d, want %dx%d", cols, rows, tc.cols, tc.rows)
			}
		})
	}
}

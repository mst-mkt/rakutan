package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"

	"github.com/blacktop/go-termimg"
)

const (
	maxImageColumns = 50
	maxImageRows    = 16
)

func ShowImage(data []byte) error {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	cols, rows := fit(img.Bounds(), imageColumns(), maxImageRows)
	slog.Debug("image", "width", img.Bounds().Dx(), "height", img.Bounds().Dy(), "cols", cols, "rows", rows)

	fmt.Print(reserve(rows))
	if err := termimg.New(img).Size(cols, rows).Print(); err != nil {
		return err
	}
	fmt.Print(release(rows))
	return nil
}

func fit(bounds image.Rectangle, maxCols, maxRows int) (cols, rows int) {
	w, h := bounds.Dx(), bounds.Dy()
	cols = maxCols
	rows = cols * h / w / 2
	if rows > maxRows {
		rows = maxRows
		cols = rows * 2 * w / h
	}
	return max(cols, 1), max(rows, 1)
}

func imageColumns() int {
	return min(terminalWidth()-2, maxImageColumns)
}

func reserve(rows int) string {
	return fmt.Sprintf("%s\x1b[%dA\x1b7", bytes.Repeat([]byte("\n"), rows), rows)
}

func release(rows int) string {
	return fmt.Sprintf("\x1b8\x1b[%dB\r\n", rows)
}

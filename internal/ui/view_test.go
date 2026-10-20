package ui

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintTableAlignsWideCharacters(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	PrintTable([][]string{
		{"\x1b[31m10/20\x1b[0m", "小テスト", "第1回"},
		{"10/27", "ドリル", "第2回"},
	})
	os.Stdout = stdout
	w.Close()
	out, _ := io.ReadAll(r)

	want := "\x1b[31m10/20\x1b[0m  小テスト  第1回\n10/27  ドリル    第2回\n"
	if string(out) != want {
		t.Errorf("table =\n%q\nwant\n%q", out, want)
	}
	if strings.Contains(stripStyle(string(out)), "\x1b") {
		t.Error("stripStyle left escapes")
	}
}

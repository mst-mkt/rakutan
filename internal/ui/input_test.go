package ui

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
	"testing"
)

const (
	left  = "\x1b[D"
	right = "\x1b[C"
	home  = "\x1b[H"
	del   = "\x1b[3~"
)

var cursorMoves = regexp.MustCompile(`\r|\x1b\[K|\x1b\[\d*D`)

func reader(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func TestEdit(t *testing.T) {
	for name, tc := range map[string]struct {
		in, initial, want string
	}{
		"typed":                   {"s1F123\r", "", "s1F123"},
		"keeps initial":           {"\r", "90", "90"},
		"edits initial":           {"\x7f8\r", "90", "98"},
		"insert in the middle":    {"ac" + left + "b\r", "", "abc"},
		"move right":              {"abc" + left + left + right + "X\r", "", "abXc"},
		"home and delete":         {"abc" + home + del + "\r", "", "bc"},
		"ctrl+a and ctrl+e":       {"bc\x01a\x05d\r", "", "abcd"},
		"ctrl+u kills before":     {"abc" + left + "\x15\r", "", "c"},
		"ctrl+k kills after":      {"abc" + left + left + "\x0b\r", "", "a"},
		"ctrl+w deletes a word":   {"foo bar\x17\r", "", "foo "},
		"backspace at start":      {home + "\x7f\r", "ab", "ab"},
		"wide characters":         {"日本" + left + "語\r", "", "日語本"},
		"unknown escape is inert": {"a\x1b[Zb\r", "", "ab"},
	} {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			got, err := edit(reader(tc.in), &out, "ID: ", tc.initial, false)
			if err != nil || got != tc.want {
				t.Errorf("edit = %q, %v, want %q", got, err, tc.want)
			}
		})
	}
}

func TestEditNeverEchoesEscapeSequences(t *testing.T) {
	var out strings.Builder
	if _, err := edit(reader("ab"+left+left+right+"\r"), &out, "ID: ", "", false); err != nil {
		t.Fatal(err)
	}
	if visible := cursorMoves.ReplaceAllString(out.String(), ""); strings.Contains(visible, "[") {
		t.Errorf("escape sequence leaked into output: %q", out.String())
	}
}

func TestEditRendersCursorPosition(t *testing.T) {
	e := editor{text: []rune("日本語"), cursor: 1}
	if got, want := e.render("> "), "\r> 日本語\x1b[K\x1b[4D"; got != want {
		t.Errorf("render = %q, want %q", got, want)
	}
}

func TestEditMasked(t *testing.T) {
	var out strings.Builder
	got, err := edit(reader("pass\x7fw\r"), &out, "パスワード: ", "", true)
	if err != nil || got != "pasw" {
		t.Errorf("edit = %q, %v, want pasw", got, err)
	}
	if strings.Contains(out.String(), "pas") {
		t.Errorf("password leaked: %q", out.String())
	}
	if !strings.HasSuffix(out.String(), "\rパスワード: ****\x1b[K\r\n") {
		t.Errorf("final render = %q", out.String())
	}
}

func TestEditAbort(t *testing.T) {
	for _, in := range []string{"ab\x03", "ab\x04", "ab"} {
		var out strings.Builder
		if _, err := edit(reader(in), &out, "ID: ", "", false); !errors.Is(err, ErrAborted) {
			t.Errorf("edit(%q): err = %v, want ErrAborted", in, err)
		}
	}
}

func TestIsYes(t *testing.T) {
	for in, want := range map[string]bool{"y": true, "YES": true, " y ": true, "n": false, "": false, "x": false} {
		if got := isYes(in); got != want {
			t.Errorf("isYes(%q) = %v, want %v", in, got, want)
		}
	}
}

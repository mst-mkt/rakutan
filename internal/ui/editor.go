package ui

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

const (
	ctrlA     = 0x01
	ctrlB     = 0x02
	ctrlC     = 0x03
	ctrlD     = 0x04
	ctrlE     = 0x05
	ctrlF     = 0x06
	ctrlK     = 0x0b
	ctrlU     = 0x15
	ctrlW     = 0x17
	escape    = 0x1b
	backspace = 0x7f
)

var escapeKeys = map[string]string{
	"C": "right", "D": "left",
	"H": "home", "1~": "home", "7~": "home",
	"F": "end", "4~": "end", "8~": "end",
	"3~": "delete",
}

type key struct {
	char rune
	name string
}

type editor struct {
	text   []rune
	cursor int
	mask   bool
}

func edit(r *bufio.Reader, w io.Writer, prompt, initial string, mask bool) (string, error) {
	e := editor{text: []rune(initial), cursor: len([]rune(initial)), mask: mask}
	fmt.Fprint(w, e.render(prompt))
	for {
		k, err := readKey(r)
		if err != nil {
			fmt.Fprint(w, "\r\n")
			return "", ErrAborted
		}

		switch k.char {
		case '\r', '\n':
			fmt.Fprint(w, "\r\n")
			return string(e.text), nil
		case ctrlC, ctrlD:
			fmt.Fprint(w, "\r\n")
			return "", ErrAborted
		}
		e = e.apply(k)
		fmt.Fprint(w, e.render(prompt))
	}
}

func readKey(r *bufio.Reader) (key, error) {
	c, _, err := r.ReadRune()
	if err != nil || c != escape {
		return key{char: c}, err
	}

	next, _, err := r.ReadRune()
	if err != nil || (next != '[' && next != 'O') {
		return key{name: "escape"}, nil
	}
	var seq []rune
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			break
		}
		seq = append(seq, c)
		if c >= 0x40 && c <= 0x7e {
			break
		}
	}
	return key{name: escapeKeys[string(seq)]}, nil
}

func (e editor) apply(k key) editor {
	switch {
	case k.name == "left" || k.char == ctrlB:
		e.cursor = max(e.cursor-1, 0)
	case k.name == "right" || k.char == ctrlF:
		e.cursor = min(e.cursor+1, len(e.text))
	case k.name == "home" || k.char == ctrlA:
		e.cursor = 0
	case k.name == "end" || k.char == ctrlE:
		e.cursor = len(e.text)
	case k.char == backspace || k.char == '\b':
		e = e.remove(max(e.cursor-1, 0), e.cursor)
	case k.name == "delete":
		e = e.remove(e.cursor, min(e.cursor+1, len(e.text)))
	case k.char == ctrlU:
		e = e.remove(0, e.cursor)
	case k.char == ctrlK:
		e = e.remove(e.cursor, len(e.text))
	case k.char == ctrlW:
		e = e.remove(wordStart(e.text, e.cursor), e.cursor)
	case k.name == "" && unicode.IsPrint(k.char):
		e.text = slices.Insert(slices.Clone(e.text), e.cursor, k.char)
		e.cursor++
	}
	return e
}

func (e editor) remove(from, to int) editor {
	e.text = slices.Delete(slices.Clone(e.text), from, to)
	e.cursor = from
	return e
}

func (e editor) render(prompt string) string {
	shown := string(e.text)
	if e.mask {
		shown = strings.Repeat("*", len(e.text))
	}
	after := runewidth.StringWidth(string([]rune(shown)[e.cursor:]))

	var b strings.Builder
	b.WriteString("\r" + prompt + cyan(shown) + "\x1b[K")
	if after > 0 {
		fmt.Fprintf(&b, "\x1b[%dD", after)
	}
	return b.String()
}

func wordStart(text []rune, cursor int) int {
	i := cursor
	for i > 0 && unicode.IsSpace(text[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(text[i-1]) {
		i--
	}
	return i
}

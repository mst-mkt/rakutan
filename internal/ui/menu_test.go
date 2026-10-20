package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/go-cmp/cmp"
)

var (
	up    = tea.KeyMsg{Type: tea.KeyUp}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	space = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}

	ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	abc  = []Choice[string]{{"A", "a"}, {"B", "b"}, {"C", "c"}}
)

func press[M tea.Model](t *testing.T, m M, msgs ...tea.Msg) M {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(M)
	}
	return m
}

func firstLine(view string) string {
	line, _, _ := strings.Cut(ansi.ReplaceAllString(view, ""), "\n")
	return strings.TrimRight(line, " ")
}

func finishKeepingFirstLine[M tea.Model](t *testing.T, m M, key tea.Msg) M {
	t.Helper()
	before := firstLine(m.View())
	m = press(t, m, key)
	if after := firstLine(m.View()); after != before {
		t.Errorf("first line changed on finish: %q -> %q", before, after)
	}
	return m
}

func assertView(t *testing.T, m tea.Model, want string) {
	t.Helper()
	if diff := cmp.Diff(want, m.View()); diff != "" {
		t.Errorf("view mismatch (-want +got):\n%s", diff)
	}
}

func TestSelect(t *testing.T) {
	m := press(t, selectModel[string]{frame: frame{title: "講義"}, choices: abc}, down, down, down, up)
	m = finishKeepingFirstLine(t, m, enter)
	assertView(t, m, "講義: B\n")
	if got := m.choices[m.cursor].Value; got != "b" {
		t.Errorf("value = %q, want b", got)
	}
}

func TestSelectAbort(t *testing.T) {
	m := press(t, selectModel[string]{frame: frame{title: "講義"}, choices: abc}, esc)
	assertView(t, m, "講義: (中断)\n")
	if !m.wasAborted() {
		t.Error("not aborted")
	}
}

func TestMultiSelect(t *testing.T) {
	m := multiSelectModel{frame: frame{title: "回答"}, choices: abc, chosen: make([]bool, 3)}
	m = press(t, m, space, down, down, space, up, space, space)
	m = finishKeepingFirstLine(t, m, enter)
	assertView(t, m, "回答: A / C\n")
	if diff := cmp.Diff([]string{"a", "c"}, m.values()); diff != "" {
		t.Errorf("values mismatch (-want +got):\n%s", diff)
	}
}

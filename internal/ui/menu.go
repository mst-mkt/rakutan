package ui

import (
	"errors"
	"log/slog"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var ErrAborted = errors.New("aborted")

func faint(s string) string {
	return "\x1b[2m" + s + "\x1b[0m"
}

type Choice[T comparable] struct {
	Label string
	Value T
}

func Select[T comparable](title string, choices []Choice[T]) (T, error) {
	var zero T
	return SelectFrom(title, choices, zero)
}

func SelectFrom[T comparable](title string, choices []Choice[T], current T) (T, error) {
	if len(choices) == 0 {
		var zero T
		return zero, errors.New("選択肢がありません")
	}
	cursor := max(0, slices.IndexFunc(choices, func(c Choice[T]) bool { return c.Value == current }))
	m, err := run(selectModel[T]{frame: frame{title: title}, choices: choices, cursor: cursor})
	slog.Debug("select", "title", title, "label", m.choices[m.cursor].Label, "aborted", m.aborted)
	return m.choices[m.cursor].Value, err
}

func MultiSelect(title string, choices []Choice[string], current []string) ([]string, error) {
	chosen := make([]bool, len(choices))
	for i, c := range choices {
		chosen[i] = slices.Contains(current, c.Value)
	}
	m, err := run(multiSelectModel{frame: frame{title: title}, choices: choices, chosen: chosen})
	slog.Debug("multiselect", "title", title, "labels", m.labels(), "aborted", m.aborted)
	return m.values(), err
}

type model interface {
	tea.Model
	wasAborted() bool
}

type frame struct {
	title   string
	done    bool
	aborted bool
}

func (f frame) wasAborted() bool {
	return f.aborted
}

func (f frame) finished() bool {
	return f.done || f.aborted
}

func (f frame) line(value string) string {
	if f.aborted {
		value = "(中断)"
	}
	return f.title + ": " + value
}

func (f frame) finish(key tea.KeyMsg) (frame, bool) {
	switch key.String() {
	case "ctrl+c", "esc":
		f.aborted = true
		return f, true
	case "enter":
		f.done = true
		return f, true
	}
	return f, false
}

func run[M model](m M) (M, error) {
	out, err := tea.NewProgram(m).Run()
	if err != nil {
		return m, err
	}
	m = out.(M)
	if m.wasAborted() {
		return m, ErrAborted
	}
	return m, nil
}

func withMenu(head string, finished bool, menu []string, help string) string {
	if finished {
		return head + "\n"
	}
	return head + "\n" + strings.Join(menu, "\n") + "\n" + faint(help) + "\n"
}

type selectModel[T comparable] struct {
	frame
	choices []Choice[T]
	cursor  int
}

func (m selectModel[T]) Init() tea.Cmd {
	return nil
}

func (m selectModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if f, quit := m.finish(key); quit {
		m.frame = f
		return m, tea.Quit
	}

	switch key.String() {
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = min(m.cursor+1, len(m.choices)-1)
	}
	return m, nil
}

func (m selectModel[T]) View() string {
	menu := make([]string, len(m.choices))
	for i, c := range m.choices {
		menu[i] = marker(i == m.cursor) + c.Label
	}
	return withMenu(m.line(m.choices[m.cursor].Label), m.finished(), menu, "↑↓ 移動  enter 決定")
}

type multiSelectModel struct {
	frame
	choices []Choice[string]
	chosen  []bool
	cursor  int
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if f, quit := m.finish(key); quit {
		m.frame = f
		return m, tea.Quit
	}

	switch key.String() {
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = min(m.cursor+1, len(m.choices)-1)
	case " ", "x":
		m.chosen = slices.Clone(m.chosen)
		m.chosen[m.cursor] = !m.chosen[m.cursor]
	}
	return m, nil
}

func (m multiSelectModel) View() string {
	menu := make([]string, len(m.choices))
	for i, c := range m.choices {
		menu[i] = marker(i == m.cursor) + checkbox(m.chosen[i]) + c.Label
	}
	return withMenu(m.line(strings.Join(m.labels(), " / ")), m.finished(), menu, "↑↓ 移動  space 選択  enter 決定")
}

func (m multiSelectModel) labels() []string {
	var labels []string
	for i, c := range m.choices {
		if m.chosen[i] {
			labels = append(labels, c.Label)
		}
	}
	return labels
}

func (m multiSelectModel) values() []string {
	var values []string
	for i, c := range m.choices {
		if m.chosen[i] {
			values = append(values, c.Value)
		}
	}
	return values
}

func marker(current bool) string {
	if current {
		return "> "
	}
	return "  "
}

func checkbox(checked bool) string {
	if checked {
		return "[x] "
	}
	return "[ ] "
}

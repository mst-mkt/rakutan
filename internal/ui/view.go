package ui

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-runewidth"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

const (
	indent       = "    "
	defaultWidth = 80
)

func PrintCover(cv *manaba.Cover) {
	fmt.Printf("\n%s\n%s\n%s\n", faint(cv.Course), bold(cv.Title), statusColor(cv.Status)(cv.Status))
}

func PrintQuestion(n, total int, q manaba.Question) {
	fmt.Printf("\n%s\n", heading(fmt.Sprintf("問題 %d/%d", n, total)))
	for line := range strings.SplitSeq(q.Text, "\n") {
		printWrapped(faint("│")+" ", line, plain)
	}
	fmt.Println()
}

func PrintAnswers(title string, qs []manaba.Question, ans manaba.Answers) {
	fmt.Printf("\n%s\n", heading(title))
	for i, q := range qs {
		fmt.Printf("%s %s\n", bold(fmt.Sprintf("%2d.", i+1)), Headline(q))
		printValues(q, ans[q.ID], green)
	}
}

func PrintReview(r *manaba.Review) {
	fmt.Printf("\n%s  %s\n", heading(r.Title), bold(r.Score))
	for i, q := range r.Questions {
		fmt.Printf("%s %s %s\n", bold(fmt.Sprintf("%2d.", i+1)), markColor(q.Mark)(q.Mark), Headline(q))
		printValues(q, q.Answer, plain)
		if len(q.Correct) > 0 && !slices.Equal(q.Correct, q.Answer) {
			fmt.Println(indent + faint("正解:"))
			printValues(q, q.Correct, green)
		}
	}
	fmt.Println()
}

func PrintTable(rows [][]string) {
	var widths []int
	for _, row := range rows {
		for i, cell := range row {
			if i == len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], displayWidth(cell))
		}
	}
	for _, row := range rows {
		var b strings.Builder
		for i, cell := range row {
			b.WriteString(cell)
			if i < len(row)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-displayWidth(cell)+2))
			}
		}
		fmt.Println(b.String())
	}
}

func PrintSaved(path string) {
	fmt.Printf("%s %s\n", green("ログを保存しました:"), faint(path))
}

func Headline(q manaba.Question) string {
	lines := strings.Split(q.Text, "\n")
	return runewidth.Truncate(lines[len(lines)-1], max(terminalWidth()-len(indent)-1, 10), "…")
}

func printValues(q manaba.Question, values []string, paint func(string) string) {
	labels := describe(q, values)
	if len(labels) == 0 {
		fmt.Println(indent + yellow("(未回答)"))
		return
	}
	for _, l := range labels {
		printWrapped(indent, l, paint)
	}
}

func printWrapped(prefix, s string, paint func(string) string) {
	width := max(terminalWidth()-displayWidth(prefix), 10)
	for line := range strings.SplitSeq(runewidth.Wrap(s, width), "\n") {
		fmt.Println(prefix + paint(line))
	}
}

func describe(q manaba.Question, values []string) []string {
	if len(q.Options) == 0 {
		return slices.DeleteFunc(slices.Clone(values), func(v string) bool { return v == "" })
	}
	var labels []string
	for _, o := range q.Options {
		if slices.Contains(values, o.Value) {
			labels = append(labels, o.Label)
		}
	}
	return labels
}

func terminalWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		return defaultWidth
	}
	return width
}

func plain(s string) string {
	return s
}

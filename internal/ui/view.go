package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

func PrintCover(cv *manaba.Cover) {
	fmt.Printf("\n%s\n%s\n%s\n\n", cv.Course, cv.Title, cv.Status)
}

func PrintQuestion(n, total int, q manaba.Question) {
	fmt.Printf("\n[%d/%d]\n%s\n\n", n, total, q.Text)
}

func PrintAnswers(qs []manaba.Question, ans manaba.Answers) {
	fmt.Println()
	for i, q := range qs {
		fmt.Printf("%2d. %s\n    %s\n", i+1, Headline(q), describe(q, ans[q.ID]))
	}
	fmt.Println()
}

func PrintReview(r *manaba.Review) {
	fmt.Printf("\n%s  %s\n\n", r.Title, r.Score)
	for i, q := range r.Questions {
		fmt.Printf("%2d. %s %s\n    回答: %s\n", i+1, q.Mark, Headline(q), describe(q, q.Answer))
		if len(q.Correct) > 0 {
			fmt.Printf("    正解: %s\n", describe(q, q.Correct))
		}
	}
	fmt.Println()
}

func Headline(q manaba.Question) string {
	lines := strings.Split(q.Text, "\n")
	return truncate(lines[len(lines)-1], 40)
}

func describe(q manaba.Question, values []string) string {
	if len(values) == 0 {
		return "(未回答)"
	}
	if len(q.Options) == 0 {
		return strings.Join(values, ", ")
	}
	var labels []string
	for _, o := range q.Options {
		if slices.Contains(values, o.Value) {
			labels = append(labels, o.Label)
		}
	}
	return strings.Join(labels, " / ")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

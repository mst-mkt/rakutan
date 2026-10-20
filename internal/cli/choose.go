package cli

import (
	"context"
	"fmt"
	"slices"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

func chooseQuiz(ctx context.Context, s *session) (string, error) {
	courses, err := s.client.Courses(ctx)
	if err != nil {
		return "", err
	}
	course, err := ui.Select("講義", courseChoices(courses))
	if err != nil {
		return "", err
	}

	quizzes, err := s.client.Quizzes(ctx, course)
	if err != nil {
		return "", err
	}
	choices := quizChoices(quizzes)
	if len(choices) == 0 {
		return "", fmt.Errorf("%s に今解ける小テストはありません", course.Title)
	}
	return ui.Select("小テスト", choices)
}

func courseChoices(courses []manaba.Course) []ui.Choice[manaba.Course] {
	var choices []ui.Choice[manaba.Course]
	for _, c := range courses {
		if c.Pending {
			choices = append(choices, ui.Choice[manaba.Course]{Label: c.Title, Value: c})
		}
	}
	return choices
}

func quizChoices(quizzes []manaba.Quiz) []ui.Choice[string] {
	var choices []ui.Choice[string]
	for _, q := range slices.SortedStableFunc(slices.Values(quizzes), byEnd) {
		if q.Answerable() {
			choices = append(choices, ui.Choice[string]{Label: fmt.Sprintf("%s 締切  %s", deadline(q.End), q.Title), Value: q.Href})
		}
	}
	return choices
}

func byEnd(a, b manaba.Quiz) int {
	return a.End.Compare(b.End)
}

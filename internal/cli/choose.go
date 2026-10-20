package cli

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

func chooseQuiz(ctx context.Context, s *session) (string, error) {
	courses, err := s.client.Courses(ctx)
	if err != nil {
		return "", err
	}
	assignments, err := s.client.Assignments(ctx)
	if err != nil {
		return "", err
	}
	now := time.Now()
	course, err := ui.Select("講義", courseChoices(courses, nextDeadlines(assignments), now))
	if err != nil {
		return "", err
	}

	quizzes, err := s.client.Quizzes(ctx, course)
	if err != nil {
		return "", err
	}
	choices := quizChoices(quizzes, now)
	if len(choices) == 0 {
		return "", fmt.Errorf("%s に今解ける小テストはありません", course.Title)
	}
	return ui.Select("小テスト", choices)
}

func nextDeadlines(assignments []manaba.Assignment) map[string]time.Time {
	next := map[string]time.Time{}
	for _, a := range assignments {
		if _, ok := manaba.QuizRef(a.Href); !ok || a.End.IsZero() {
			continue
		}
		if t, seen := next[a.Course]; !seen || a.End.Before(t) {
			next[a.Course] = a.End
		}
	}
	return next
}

func courseChoices(courses []manaba.Course, next map[string]time.Time, now time.Time) []ui.Choice[manaba.Course] {
	var choices []ui.Choice[manaba.Course]
	for _, c := range courses {
		if !c.Pending {
			continue
		}
		choice := ui.Choice[manaba.Course]{Label: c.Title, Value: c}
		if end, ok := next[c.Title]; ok {
			choice.Detail = ui.Deadline(end, now)
		}
		choices = append(choices, choice)
	}
	return choices
}

func quizChoices(quizzes []manaba.Quiz, now time.Time) []ui.Choice[string] {
	var choices []ui.Choice[string]
	for _, q := range slices.SortedStableFunc(slices.Values(quizzes), byEnd) {
		if q.Answerable() {
			choices = append(choices, ui.Choice[string]{Label: q.Title, Detail: ui.Deadline(q.End, now), Value: q.Href})
		}
	}
	return choices
}

func byEnd(a, b manaba.Quiz) int {
	return a.End.Compare(b.End)
}

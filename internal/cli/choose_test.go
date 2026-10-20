package cli

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

var now = time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)

func labels[T comparable](choices []ui.Choice[T]) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Label
	}
	return out
}

func TestNextDeadlines(t *testing.T) {
	got := nextDeadlines([]manaba.Assignment{
		{Href: "course_1_query_2", Course: "A", End: now.Add(48 * time.Hour)},
		{Href: "course_1_query_3", Course: "A", End: now.Add(5 * time.Hour)},
		{Href: "course_2_query_4", Course: "B", End: now.Add(10 * 24 * time.Hour)},
		{Href: "course_3_drill_5", Course: "C", End: now.Add(time.Hour)},
		{Href: "course_4_query_6", Course: "D"},
	})
	want := map[string]time.Time{"A": now.Add(5 * time.Hour), "B": now.Add(10 * 24 * time.Hour)}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCourseChoices(t *testing.T) {
	courses := []manaba.Course{
		{Title: "A", Pending: true},
		{Title: "B"},
		{Title: "C", Pending: true},
	}
	got := labels(courseChoices(courses, map[string]time.Time{"A": now.Add(5 * time.Hour)}, now))
	want := []string{"A  10/20 15:00 締切 (あと 5 時間)", "C"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestQuizChoices(t *testing.T) {
	quizzes := []manaba.Quiz{
		{Title: "第3回", Href: "course_1_query_3", Status: "受付中 未提出", End: now.Add(4 * 24 * time.Hour)},
		{Title: "第2回", Href: "course_1_query_2", Status: "受付中 提出済み", End: now.Add(time.Hour)},
		{Title: "第4回", Href: "course_1_query_4", Status: "受付中 未提出", End: now.Add(2 * time.Hour)},
		{Title: "第5回", Href: "course_1_query_5", Status: "受付開始待ち", End: now.Add(9 * 24 * time.Hour)},
	}
	choices := quizChoices(quizzes, now)
	want := []string{"10/20 12:00 締切 (あと 2 時間)  第4回", "10/24 10:00 締切 (あと 4 日)  第3回"}
	if diff := cmp.Diff(want, labels(choices)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if choices[0].Value != "course_1_query_4" {
		t.Errorf("first value = %s", choices[0].Value)
	}
}

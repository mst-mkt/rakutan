package cli

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

var now = time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNextDeadlines(t *testing.T) {
	got := nextDeadlines([]manaba.Assignment{
		{Href: "course_1_query_2", Course: "A", End: now.Add(48 * time.Hour)},
		{Href: "course_1_query_3", Course: "A", End: now.Add(5 * time.Hour)},
		{Href: "course_2_query_4", Course: "B", End: now.Add(10 * 24 * time.Hour)},
		{Href: "course_3_drill_5", Course: "C", End: now.Add(time.Hour)},
		{Href: "course_4_query_6", Course: "D"},
	})
	assertEqual(t, got, map[string]time.Time{"A": now.Add(5 * time.Hour), "B": now.Add(10 * 24 * time.Hour)})
}

func TestCourseChoices(t *testing.T) {
	a := manaba.Course{Title: "A", Pending: true}
	c := manaba.Course{Title: "C", Pending: true}
	got := courseChoices([]manaba.Course{a, {Title: "B"}, c}, map[string]time.Time{"A": now.Add(5 * time.Hour)}, now)
	assertEqual(t, got, []ui.Choice[manaba.Course]{
		{Label: "A", Detail: "10/20 15:00 締切 (あと 5 時間)", Value: a},
		{Label: "C", Value: c},
	})
}

func TestQuizChoices(t *testing.T) {
	got := quizChoices([]manaba.Quiz{
		{Title: "第3回", Href: "course_1_query_3", Status: "受付中 未提出", End: now.Add(4 * 24 * time.Hour)},
		{Title: "第2回", Href: "course_1_query_2", Status: "受付中 提出済み", End: now.Add(time.Hour)},
		{Title: "第4回", Href: "course_1_query_4", Status: "受付中 未提出", End: now.Add(2 * time.Hour)},
		{Title: "第5回", Href: "course_1_query_5", Status: "受付開始待ち", End: now.Add(9 * 24 * time.Hour)},
	}, now)
	assertEqual(t, got, []ui.Choice[string]{
		{Label: "第4回", Detail: "10/20 12:00 締切 (あと 2 時間)", Value: "course_1_query_4"},
		{Label: "第3回", Detail: "10/24 10:00 締切 (あと 4 日)", Value: "course_1_query_3"},
	})
}

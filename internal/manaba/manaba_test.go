package manaba

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func loadPage(t *testing.T, name, ref string) *Page {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := NewPage(BaseURL+ref, f)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func parse[T any](t *testing.T, parse func(*Page) (T, error), name, ref string) T {
	t.Helper()
	v, err := parse(loadPage(t, name, ref))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty(), cmp.AllowUnexported(Form{}, Review{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

var dummyQuestions = []Question{
	{ID: "qid1", Kind: KindText, Text: "共通の指示\nQ1;本文1<.>"},
	{ID: "qid2", Kind: KindCheckbox, Text: "Q2;複数選べ", Options: []Option{{"2", "選択肢B"}, {"1", "選択肢A"}}},
	{ID: "qid3", Kind: KindSelect, Text: "Q3;下の図", Options: []Option{{"2", "語句B"}, {"1", "語句A"}},
		Images: []string{BaseURL + "course_1_query_2__3/fig.png?view=inline"}},
	{ID: "qid4", Kind: KindRadio, Text: "Q4;一つ選べ", Options: []Option{{"1", "はい"}, {"2", "いいえ"}}},
}

func TestParseSheet(t *testing.T) {
	s := parse(t, ParseSheet, "quiz.html", "course_1_query_2_1p1?refresh=AAAA")
	assertEqual(t, s.Title, "ダミー小テスト")
	assertEqual(t, s.Questions, dummyQuestions)
}

func TestSheetFill(t *testing.T) {
	s := parse(t, ParseSheet, "quiz.html", "course_1_query_2_1p1")
	assertEqual(t, s.fill(Answers{"qid1": {"42"}, "qid2": {"1", "2"}}, saveButton), Form{
		Action:    BaseURL + "course_1_query_2_1p1",
		Multipart: true,
		Values: url.Values{
			"QueryStartUnix": {"1700000000"},
			"TargetQIDs":     {"X,qid1,qid2,qid3,qid4"},
			"manaba-form":    {"1"},
			"SessionValue1":  {"dummy-token"},
			"SessionValue":   {"@1"},
			"qid1":           {"42"},
			"qid2":           {"1", "2"},
			"qid3":           {""},
			saveButton.Name:  {saveButton.Value},
		},
	})
}

func TestFormWithKeepsOriginal(t *testing.T) {
	f := Form{Values: url.Values{"a": {"1"}}}
	_ = f.With(url.Values{"a": {"2"}, "b": {"3"}})
	assertEqual(t, f.Values, url.Values{"a": {"1"}})
}

var dummyReview = &Review{
	URL:         BaseURL + "course_1_query_2",
	Course:      "ダミーコース",
	Title:       "第1回「ダミー」",
	SubmittedAt: "2026-10-20 07:43",
	Score:       "8 / 8",
	Questions: []Question{
		{ID: "qid1", Kind: KindText, Text: "Q1;本文1", Answer: []string{"42"}, Mark: "○", Score: "3"},
		{ID: "qid2", Kind: KindCheckbox, Text: "Q2;複数選べ", Options: []Option{{"2", "選択肢B"}, {"1", "選択肢A"}},
			Answer: []string{"1"}, Mark: "100%正解", Score: "5"},
	},
	answerKey: BaseURL + "course_1_query_2?answer=1&answershuffle=response",
}

func TestParseReview(t *testing.T) {
	assertEqual(t, parse(t, ParseReview, "review.html", "course_1_query_2?ReviewMode=1"), dummyReview)
}

func TestParseAnswerKey(t *testing.T) {
	assertEqual(t, parse(t, ParseAnswerKey, "answer_key.html", "course_1_query_2?answer=1&answershuffle=response"), []Question{
		{ID: "qid1", Kind: KindText, Text: "Q1;本文1", Answer: []string{"90"}, Score: "3"},
		{ID: "qid2", Kind: KindCheckbox, Text: "Q2;複数選べ", Options: []Option{{"2", "選択肢B"}, {"1", "選択肢A"}},
			Answer: []string{"2"}, Score: "5"},
	})
}

func TestReviewWithAnswerKey(t *testing.T) {
	key := parse(t, ParseAnswerKey, "answer_key.html", "course_1_query_2?answer=1")
	got := dummyReview.withAnswerKey(key)
	assertEqual(t, got.Questions[0].Correct, []string{"90"})
	assertEqual(t, got.Questions[0].Points, "3")
	assertEqual(t, got.Questions[1].Correct, []string{"2"})
	assertEqual(t, dummyReview.Questions[0].Correct, []string(nil))
}

func TestParseReviewNotSubmitted(t *testing.T) {
	if _, err := ParseReview(loadPage(t, "quiz.html", "course_1_query_2")); !errors.Is(err, ErrNotSubmitted) {
		t.Errorf("err = %v, want %v", err, ErrNotSubmitted)
	}
}

func TestParseAssignments(t *testing.T) {
	assertEqual(t, parse(t, ParseAssignments, "assignments.html", "home_library_query"), []Assignment{
		{
			Kind: "小テスト", Title: "第1回「ダミー」", Href: "course_1_query_2", Course: "ダミーコース",
			Start: time.Date(2026, 10, 1, 9, 30, 0, 0, jst), End: time.Date(2026, 10, 20, 17, 0, 0, 0, jst),
		},
		{
			Kind: "ドリル", Title: "ダミードリル", Href: "course_3_drill_4", Course: "お知らせ",
			Start: time.Date(2023, 4, 8, 10, 37, 0, 0, jst),
		},
	})
}

func TestParseCourses(t *testing.T) {
	assertEqual(t, parse(t, ParseCourses, "courses.html", "home_course"), []Course{
		{Title: "ダミーコース", Href: "course_1", Year: "2026", Term: "秋学期 / 火6", Teacher: "ダミー 教員", Pending: true},
		{Title: "お知らせ", Href: "course_3", Year: "2999", Teacher: "事務課"},
	})
}

func TestParseQuizzes(t *testing.T) {
	assertEqual(t, parse(t, ParseQuizzes, "quizzes.html", "course_1_query"), []Quiz{
		{
			Title: "第1回「ダミー」", Href: "course_1_query_2", Status: "受付中 提出済み",
			Start: time.Date(2026, 10, 1, 9, 30, 0, 0, jst), End: time.Date(2026, 10, 20, 17, 0, 0, 0, jst),
		},
		{
			Title: "第2回：ダミー", Href: "course_1_query_4", Status: "受付開始待ち",
			Start: time.Date(2026, 10, 22, 9, 30, 0, 0, jst), End: time.Date(2026, 11, 24, 17, 0, 0, 0, jst),
		},
	})
}

func TestQuizAnswerable(t *testing.T) {
	for status, want := range map[string]bool{
		"受付中 未提出":   true,
		"受付中 提出済み":  false,
		"受付開始待ち":    false,
		"受付終了 未提出":  false,
		"受付終了 提出済み": false,
	} {
		if got := (Quiz{Href: "course_1_query_2", Status: status}).Answerable(); got != want {
			t.Errorf("%s: Answerable = %v, want %v", status, got, want)
		}
	}
	if (Quiz{Href: "course_3_drill_4", Status: "受付中 未提出"}).Answerable() {
		t.Error("drill should not be answerable")
	}
}

func TestAuthed(t *testing.T) {
	for name, tc := range map[string]struct {
		html string
		want error
	}{
		"logged in":          {`<a href="logout">ログアウト</a>`, nil},
		"local login form":   {`<form action="login"><input name="userid"><input type="password" name="password"></form>`, ErrLoginRequired},
		"another login page": {`<a href="https://sso.example.com/login">ログイン</a>`, ErrLoginRequired},
	} {
		t.Run(name, func(t *testing.T) {
			p, err := NewPage(BaseURL+"home", strings.NewReader(tc.html))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := authed(p, nil); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestQuizRef(t *testing.T) {
	for in, want := range map[string]string{
		"https://www.ace.toyo.ac.jp/ct/course_1_query_2?ReviewMode=1": "course_1_query_2",
		"course_1_query_2_1p1": "course_1_query_2",
		"course_3_drill_4":     "",
	} {
		got, ok := QuizRef(in)
		assertEqual(t, got, want)
		assertEqual(t, ok, want != "")
	}
}

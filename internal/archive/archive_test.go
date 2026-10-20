package archive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

type fakeDownloader map[string][]byte

func (f fakeDownloader) Download(_ context.Context, ref string) ([]byte, error) {
	return f[ref], nil
}

func review(images ...string) *manaba.Review {
	return &manaba.Review{
		URL:         manaba.BaseURL + "course_1_query_2",
		Course:      "ダミー/コース",
		Title:       "第1回「ダミー」",
		SubmittedAt: "2026-10-20 07:43",
		Score:       "8 / 8",
		Questions: []manaba.Question{
			{ID: "qid1", Kind: manaba.KindText, Text: "共通の指示\nQ1;本文1", Answer: []string{"42"}, Mark: "×", Score: "0",
				Correct: []string{"90"}, Points: "3", Images: images},
			{ID: "qid2", Kind: manaba.KindCheckbox, Text: "Q2;複数選べ",
				Options: []manaba.Option{{Value: "2", Label: "選択肢B"}, {Value: "1", Label: "選択肢A"}},
				Answer:  []string{"1"}, Mark: "100%正解", Score: "5", Correct: []string{"1"}, Points: "5"},
		},
	}
}

func TestRender(t *testing.T) {
	var b strings.Builder
	err := Render(&b, review("https://example.com/fig.png?view=inline"),
		map[string]string{"https://example.com/fig.png?view=inline": "images/fig.png"})
	if err != nil {
		t.Fatal(err)
	}
	want := `---
course: "ダミー/コース"
title: "第1回「ダミー」"
url: ` + manaba.BaseURL + `course_1_query_2
submitted_at: "2026-10-20 07:43"
score: "8 / 8"
---

# 第1回「ダミー」

## qid1

共通の指示
Q1;本文1

![](images/fig.png)

回答: 42
正解: 90

判定: × (0 / 3 点)

## qid2

Q2;複数選べ

- [ ] 選択肢B
- [x] 選択肢A (正解)

判定: 100%正解 (5 / 5 点)
`
	if diff := cmp.Diff(want, b.String()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSave(t *testing.T) {
	src := "https://example.com/fig.png?view=inline"
	for name, r := range map[string]*manaba.Review{
		"without images": review(),
		"with images":    review(src),
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			file, err := Save(t.Context(), fakeDownloader{src: []byte("png")}, root, r)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "ダミー／コース")
			if want := filepath.Join(dir, "第1回「ダミー」.md"); file != want {
				t.Errorf("file = %s, want %s", file, want)
			}
			if _, err := os.Stat(file); err != nil {
				t.Error(err)
			}
			if len(r.Questions[0].Images) == 0 {
				return
			}
			data, err := os.ReadFile(filepath.Join(dir, "images", "第1回「ダミー」-qid1-1.png"))
			if err != nil || string(data) != "png" {
				t.Errorf("image = %q, %v", data, err)
			}
		})
	}
}

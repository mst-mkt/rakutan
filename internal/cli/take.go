package cli

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/cli/browser"

	"github.com/mst-mkt/rakutan/internal/archive"
	"github.com/mst-mkt/rakutan/internal/assist"
	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

type takeCmd struct {
	Quiz string `arg:"" optional:"" help:"小テストの URL (省略すると一覧から選ぶ)"`
	Mode string `enum:",browser,terminal,ai" default:"" placeholder:"browser|terminal|ai" help:"回答方法 (省略すると開始前に聞く)"`
}

const (
	browserMode  = "browser"
	terminalMode = "terminal"
	aiMode       = "ai"
)

var modes = []ui.Choice[string]{
	{Label: "ブラウザで開く", Value: browserMode},
	{Label: "ターミナルで回答する", Value: terminalMode},
	{Label: "AI の回答を元に回答する", Value: aiMode},
}

type fallback int

const (
	retryFallback fallback = iota
	manualFallback
	abortFallback
)

var fallbacks = []ui.Choice[fallback]{
	{Label: "もう一度 AI に頼む", Value: retryFallback},
	{Label: "自分で回答する", Value: manualFallback},
	{Label: "中断する", Value: abortFallback},
}

type action int

const (
	submitAction action = iota
	editAction
	reviewAction
	saveAction
	quitAction
)

var actions = []ui.Choice[action]{
	{Label: "提出する", Value: submitAction},
	{Label: "問題を選んで直す", Value: editAction},
	{Label: "全問を順に確認する", Value: reviewAction},
	{Label: "一時保存して終了", Value: saveAction},
	{Label: "保存せずに終了", Value: quitAction},
}

type attempt struct {
	s       *session
	sheet   *manaba.Sheet
	answers manaba.Answers
}

func (c takeCmd) Run(ctx context.Context, s *session) error {
	ref, err := c.quizRef(ctx, s)
	if err != nil {
		return err
	}
	cv, err := s.client.Cover(ctx, ref)
	if err != nil {
		return err
	}

	slog.Info("cover", "ref", ref, "title", cv.Title, "status", cv.Status, "startable", cv.Startable())
	ui.PrintCover(cv)
	if cv.Submitted {
		r, err := s.client.Review(ctx, ref)
		if err != nil {
			return err
		}
		return showResult(ctx, s, r)
	}
	if !cv.Startable() {
		return fmt.Errorf("この小テストは開始できません (%s)", cv.Status)
	}
	mode, err := c.chooseMode()
	if err != nil {
		return err
	}
	slog.Info("mode", "mode", mode)
	if mode == browserMode {
		return browser.OpenURL(cv.URL)
	}
	if mode == aiMode {
		ui.Notice(aiCaution)
	}

	sheet, err := s.client.Start(ctx, cv)
	if err != nil {
		return err
	}
	slog.Info("sheet", "questions", len(sheet.Questions))
	a := &attempt{s: s, sheet: sheet, answers: manaba.AnswersOf(sheet.Questions)}
	if err := a.fill(ctx, mode == aiMode); err != nil {
		return err
	}
	return a.menu(ctx)
}

const aiCaution = "注意: AI の回答案は、AI の利用が認められている講義やアンケートのための機能です。" +
	"禁止されている講義で使うと、成績や単位に影響するおそれがあります。各講義の方針を確認してください。"

func (c takeCmd) chooseMode() (string, error) {
	if c.Mode != "" {
		return c.Mode, nil
	}
	ui.Hint("ターミナルで回答すると、選んだ時点で経過時間の計測が始まります")
	return ui.Select("回答方法", modes)
}

func (c takeCmd) quizRef(ctx context.Context, s *session) (string, error) {
	if c.Quiz == "" {
		return chooseQuiz(ctx, s)
	}
	ref, ok := manaba.QuizRef(c.Quiz)
	if !ok {
		return "", fmt.Errorf("小テストの URL ではありません: %s", c.Quiz)
	}
	return ref, nil
}

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

func questionChoices(qs []manaba.Question) []ui.Choice[int] {
	choices := make([]ui.Choice[int], len(qs))
	for i, q := range qs {
		choices[i] = ui.Choice[int]{Label: fmt.Sprintf("%d. %s", i+1, ui.Headline(q)), Value: i}
	}
	return choices
}

func (a *attempt) fill(ctx context.Context, useAI bool) error {
	if useAI {
		suggested, err := a.suggest(ctx)
		if err != nil || suggested {
			return err
		}
	}
	return a.answerAll(ctx)
}

func (a *attempt) answerAll(ctx context.Context) error {
	for i := range a.sheet.Questions {
		if err := a.answer(ctx, i); err != nil {
			return err
		}
	}
	return nil
}

func (a *attempt) suggest(ctx context.Context) (bool, error) {
	for {
		var suggestion manaba.Answers
		err := ui.Wait("AI が回答を考えています", func() (err error) {
			suggestion, err = assist.Suggest(ctx, a.s.client, a.sheet, archive.CourseDir(archiveDir, a.sheet.Course))
			return err
		})
		if err == nil {
			for id, v := range suggestion {
				if len(v) > 0 {
					a.answers[id] = v
				}
			}
			return true, nil
		}

		warn(err, "AI の回答案を受け取れませんでした")
		next, err := ui.Select("どうしますか", fallbacks)
		if err != nil {
			return false, err
		}
		switch next {
		case manualFallback:
			return false, nil
		case abortFallback:
			return false, ui.ErrAborted
		}
	}
}

func (a *attempt) answer(ctx context.Context, i int) error {
	q := a.sheet.Questions[i]
	ui.PrintQuestion(i+1, len(a.sheet.Questions), q)
	for _, src := range q.Images {
		data, err := a.s.client.Download(ctx, src)
		if err == nil {
			err = ui.ShowImage(data)
		}
		warn(err, "画像を表示できません")
	}

	v, err := ui.Ask(q, a.answers[q.ID])
	if err != nil {
		return err
	}
	a.answers[q.ID] = v
	return nil
}

func (a *attempt) menu(ctx context.Context) error {
	for {
		ui.PrintAnswers("回答一覧", a.sheet.Questions, a.answers)
		act, err := ui.Select("どうしますか", actions)
		if err != nil {
			return err
		}

		switch act {
		case submitAction:
			done, err := a.submit(ctx)
			if err != nil || done {
				return err
			}
		case editAction:
			i, err := ui.Select("直す問題", questionChoices(a.sheet.Questions))
			if err != nil {
				return err
			}
			if err := a.answer(ctx, i); err != nil {
				return err
			}
		case reviewAction:
			if err := a.answerAll(ctx); err != nil {
				return err
			}
		case saveAction:
			if err := a.s.client.Save(ctx, a.sheet, a.answers); err != nil {
				return err
			}
			ui.Done("一時保存しました")
			return nil
		case quitAction:
			return nil
		}
	}
}

func (a *attempt) submit(ctx context.Context) (bool, error) {
	cf, err := a.s.client.Confirm(ctx, a.sheet, a.answers)
	if err != nil {
		return false, err
	}

	ui.PrintAnswers("サーバーが受け付けた回答", cf.Questions, manaba.AnswersOf(cf.Questions))
	ok, err := ui.Confirm("この内容で提出しますか?")
	if err != nil {
		return false, err
	}
	if !ok {
		a.sheet, err = a.s.client.Back(ctx, cf)
		return false, err
	}

	r, err := a.s.client.Finish(ctx, cf)
	if err != nil {
		return false, err
	}
	if err := showResult(ctx, a.s, r); err != nil {
		return true, err
	}

	open, err := ui.Confirm("ブラウザで結果のページを開きますか?")
	if err != nil || !open {
		return true, err
	}
	return true, browser.OpenURL(r.URL + "?ReviewMode=1")
}

func showResult(ctx context.Context, s *session, r *manaba.Review) error {
	ui.PrintReview(r)
	file, err := saveLog(ctx, s, r)
	if err != nil {
		return err
	}
	ui.PrintSaved(file)
	return nil
}

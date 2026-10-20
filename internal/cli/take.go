package cli

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

type takeCmd struct {
	Quiz string `arg:"" optional:"" help:"小テストの URL (省略すると一覧から選ぶ)"`
}

type action int

const (
	submitAction action = iota
	editAction
	saveAction
	quitAction
)

var actions = []ui.Choice[action]{
	{Label: "提出する", Value: submitAction},
	{Label: "問題を選んで直す", Value: editAction},
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
	if ok, err := ui.Confirm("開始すると経過時間の計測が始まります。開始しますか?"); err != nil || !ok {
		return err
	}

	sheet, err := s.client.Start(ctx, cv)
	if err != nil {
		return err
	}
	slog.Info("sheet", "questions", len(sheet.Questions))
	a := &attempt{s: s, sheet: sheet, answers: manaba.AnswersOf(sheet.Questions)}
	for i := range sheet.Questions {
		if err := a.answer(ctx, i); err != nil {
			return err
		}
	}
	return a.menu(ctx)
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
		ui.PrintAnswers(a.sheet.Questions, a.answers)
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
		case saveAction:
			if err := a.s.client.Save(ctx, a.sheet, a.answers); err != nil {
				return err
			}
			fmt.Println("一時保存しました")
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

	fmt.Println("\nサーバーに送った回答:")
	ui.PrintAnswers(cf.Questions, manaba.AnswersOf(cf.Questions))
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
	return true, showResult(ctx, a.s, r)
}

func showResult(ctx context.Context, s *session, r *manaba.Review) error {
	ui.PrintReview(r)
	file, err := saveLog(ctx, s, r)
	if err != nil {
		return err
	}
	fmt.Printf("ログを保存しました: %s\n", file)
	return nil
}

package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cli/browser"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/ui"
)

type answerMode string

const (
	browserMode  answerMode = "browser"
	terminalMode answerMode = "terminal"
	aiMode       answerMode = "ai"
)

var modes = []ui.Choice[answerMode]{
	{Label: "ブラウザで開く", Value: browserMode},
	{Label: "ターミナルで回答する", Value: terminalMode},
	{Label: "AI の回答を元に回答する", Value: aiMode},
}

const aiCaution = "注意: AI の回答案は、AI の利用が認められている講義やアンケートのための機能です。" +
	"禁止されている講義で使うと、成績や単位に影響するおそれがあります。各講義の方針を確認してください。"

type takeCmd struct {
	Quiz string     `arg:"" optional:"" help:"小テストの URL (省略すると一覧から選ぶ)"`
	Mode answerMode `enum:",browser,terminal,ai" default:"" placeholder:"browser|terminal|ai" help:"回答方法 (省略すると開始前に聞く)"`
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
		if err := browser.OpenURL(cv.URL); err != nil {
			return err
		}
		ui.Done("ブラウザで開きました")
		return nil
	}
	if mode == aiMode {
		ui.Notice(aiCaution)
	}

	a, err := start(ctx, s, cv)
	if err != nil {
		return err
	}
	if err := a.fill(ctx, mode); err != nil {
		return err
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

func (c takeCmd) chooseMode() (answerMode, error) {
	if c.Mode != "" {
		return c.Mode, nil
	}
	ui.Hint("ターミナルで回答すると、選んだ時点で経過時間の計測が始まります")
	return ui.Select("回答方法", modes)
}

package cli

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"

	"github.com/mst-mkt/rakutan/internal/trace"
	"github.com/mst-mkt/rakutan/internal/ui"
)

const keepRuns = 10

var debugDir = filepath.Join(xdg.StateHome, "rakutan", "debug")

type cli struct {
	Take   takeCmd   `cmd:"" default:"withargs" help:"小テストを選んで解く (サブコマンド省略時の動作)"`
	Login  loginCmd  `cmd:"" help:"ログインして認証情報を keyring に保存する"`
	Logout logoutCmd `cmd:"" help:"keyring の認証情報と cookie を削除する"`
	Ls     lsCmd     `cmd:"" help:"未提出の課題を締切順に表示する"`
	Log    logCmd    `cmd:"" help:"提出済みの小テストを Markdown で保存する"`
}

func Run() {
	var c cli
	transport := http.DefaultTransport
	k := kong.Parse(&c,
		kong.Name("rakutan"),
		kong.Description("Toyo Net Ace の小テストをターミナルで解く"),
		kong.BindTo(context.Background(), (*context.Context)(nil)),
		kong.BindToProvider(func() (*session, error) { return openSession(transport) }),
	)

	slog.SetDefault(slog.New(slog.DiscardHandler))
	tr := openTrace(k)
	if tr != nil {
		slog.SetDefault(tr.Logger())
		transport = tr.Transport(transport)
	}

	code := run(k)
	if tr != nil {
		warn(tr.Close(), "デバッグログを閉じられません")
	}
	os.Exit(code)
}

func openTrace(k *kong.Context) *trace.Trace {
	if v := os.Getenv("RAKUTAN_DEBUG"); v == "" || v == "0" {
		return nil
	}

	command, _, _ := strings.Cut(k.Command(), " ")
	tr, err := trace.Open(debugDir, command, keepRuns)
	if err != nil {
		warn(err, "デバッグログを開けません")
		return nil
	}
	return tr
}

func run(k *kong.Context) int {
	slog.Info("command", "name", k.Command(), "args", os.Args[1:])

	switch err := k.Run(); {
	case err == nil:
		return 0
	case errors.Is(err, ui.ErrAborted):
		slog.Info("aborted")
		ui.Notice("中断しました")
		return 130
	default:
		slog.Error("failed", "err", err)
		ui.Error(err)
		return 1
	}
}

func warn(err error, msg string) {
	if err != nil {
		slog.Warn(msg, "err", err)
		ui.Warn(msg, err)
	}
}

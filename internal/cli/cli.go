package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/mst-mkt/rakutan/internal/ui"
)

type cli struct {
	Take   takeCmd   `cmd:"" default:"withargs" help:"小テストを選んで解く (サブコマンド省略時の動作)"`
	Login  loginCmd  `cmd:"" help:"ログインして認証情報を keyring に保存する"`
	Logout logoutCmd `cmd:"" help:"keyring の認証情報と cookie を削除する"`
	Ls     lsCmd     `cmd:"" help:"未提出の課題を締切順に表示する"`
	Log    logCmd    `cmd:"" help:"提出済みの小テストを Markdown で保存する"`
}

func Run() {
	var c cli
	k := kong.Parse(&c,
		kong.Name("rakutan"),
		kong.Description("Toyo Net Ace の小テストをターミナルで解く"),
		kong.BindTo(context.Background(), (*context.Context)(nil)),
		kong.BindToProvider(openSession),
	)

	err := k.Run()
	if errors.Is(err, ui.ErrAborted) {
		fmt.Fprintln(os.Stderr, "中断しました")
		os.Exit(130)
	}
	k.FatalIfErrorf(err)
}

func warn(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	}
}

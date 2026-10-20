package cli

import (
	"context"
	"fmt"
	"os"
)

type loginCmd struct{}

func (loginCmd) Run(ctx context.Context, s *session) error {
	if err := s.loginInteractively(ctx); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "ログインしました")
	return nil
}

type logoutCmd struct{}

func (logoutCmd) Run(s *session) error {
	return s.logout()
}

package cli

import (
	"context"

	"github.com/mst-mkt/rakutan/internal/ui"
)

type loginCmd struct{}

func (loginCmd) Run(ctx context.Context, s *session) error {
	if err := s.loginInteractively(ctx); err != nil {
		return err
	}
	ui.Done("ログインしました")
	return nil
}

type logoutCmd struct{}

func (logoutCmd) Run(s *session) error {
	return s.logout()
}

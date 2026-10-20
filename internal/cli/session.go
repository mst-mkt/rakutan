package cli

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/adrg/xdg"

	"github.com/mst-mkt/rakutan/internal/manaba"
	"github.com/mst-mkt/rakutan/internal/secret"
	"github.com/mst-mkt/rakutan/internal/ui"
)

type credential struct {
	UserID   string `json:"userid"`
	Password string `json:"password"`
}

var (
	archiveDir   = filepath.Join(xdg.DataHome, "rakutan", "archive")
	credentials  = secret.New[credential]("credential")
	sessionStore = secret.WithFileFallback[[]manaba.Cookie]("session", filepath.Join(xdg.StateHome, "rakutan", "cookies.json"))
)

type session struct {
	client *manaba.Client
	jar    *manaba.Jar
}

func openSession(transport http.RoundTripper) (*session, error) {
	cookies, err := sessionStore.Load()
	if err != nil && !errors.Is(err, secret.ErrNotFound) {
		warn(err, "保存したセッションを読めません")
	}
	slog.Debug("session", "restored", len(cookies) > 0)

	s := &session{jar: manaba.NewJar(cookies)}
	s.client = manaba.New(&http.Client{Jar: s.jar, Transport: transport}, s.login)
	return s, nil
}

func (s *session) login(ctx context.Context) error {
	cred, err := credentials.Load()
	if err != nil {
		slog.Info("login", "credential", "prompt", "keyring", err)
		return s.loginInteractively(ctx)
	}
	slog.Info("login", "credential", "keyring")
	return s.loginWith(ctx, cred)
}

func (s *session) loginInteractively(ctx context.Context) error {
	userID, password, err := ui.Credential()
	if err != nil {
		return err
	}
	cred := credential{UserID: userID, Password: password}
	if err := s.loginWith(ctx, cred); err != nil {
		return err
	}

	warn(credentials.Save(cred), "keyring が使えないため、認証情報は保存しません")
	return nil
}

func (s *session) loginWith(ctx context.Context, cred credential) error {
	if err := s.client.Login(ctx, cred.UserID, cred.Password); err != nil {
		return err
	}
	slog.Info("login succeeded", "userid", cred.UserID)
	return sessionStore.Save(s.jar.Export())
}

func (s *session) logout() error {
	return errors.Join(credentials.Delete(), sessionStore.Delete())
}

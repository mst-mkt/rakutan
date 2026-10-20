package manaba

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

var (
	ErrLoginRequired = errors.New("manaba: login required")
	ErrLoginFailed   = errors.New("manaba: wrong user ID or password")
)

const loginField = `input[name="userid"]`

func (c *Client) Login(ctx context.Context, userID, password string) error {
	p, err := c.fetch(ctx, http.MethodGet, "login", nil, "")
	if err != nil {
		return err
	}
	f, err := findForm(p, loginField)
	if err != nil {
		return err
	}

	body, contentType, err := f.With(url.Values{
		"userid":   {userID},
		"password": {password},
		"login":    {"ログイン"},
	}).encode()
	if err != nil {
		return err
	}
	res, err := c.fetch(ctx, http.MethodPost, f.Action, body, contentType)
	if err != nil {
		return err
	}
	if res.Doc.Find(loginField).Length() > 0 {
		return ErrLoginFailed
	}

	_, err = authed(c.fetch(ctx, http.MethodGet, "home", nil, ""))
	if errors.Is(err, ErrLoginRequired) {
		return ErrLoginFailed
	}
	return err
}

func authed(p *Page, err error) (*Page, error) {
	switch {
	case err != nil:
		return nil, err
	case p.Doc.Find(`a[href="logout"]`).Length() == 0:
		return nil, ErrLoginRequired
	default:
		return p, nil
	}
}

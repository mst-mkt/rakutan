package manaba

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	BaseURL   = "https://www.ace.toyo.ac.jp/local/"
	userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"
)

type Client struct {
	http   *http.Client
	base   *url.URL
	reauth func(context.Context) error
}

func New(hc *http.Client, reauth func(context.Context) error) *Client {
	base, _ := url.Parse(BaseURL)
	return &Client{http: hc, base: base, reauth: reauth}
}

func (c *Client) Get(ctx context.Context, ref string) (*Page, error) {
	return retryAfterLogin(ctx, c, func() (*Page, error) {
		return authed(c.fetch(ctx, http.MethodGet, ref, nil, ""))
	})
}

func (c *Client) Submit(ctx context.Context, f Form) (*Page, error) {
	body, contentType, err := f.encode()
	if err != nil {
		return nil, err
	}
	return authed(c.fetch(ctx, http.MethodPost, f.Action, body, contentType))
}

func (c *Client) Download(ctx context.Context, ref string) ([]byte, error) {
	return retryAfterLogin(ctx, c, func() ([]byte, error) {
		res, err := c.do(ctx, http.MethodGet, ref, nil, "")
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
			return nil, ErrLoginRequired
		}
		return io.ReadAll(res.Body)
	})
}

func retryAfterLogin[T any](ctx context.Context, c *Client, f func() (T, error)) (T, error) {
	v, err := f()
	if !errors.Is(err, ErrLoginRequired) || c.reauth == nil {
		return v, err
	}

	if err := c.reauth(ctx); err != nil {
		return v, err
	}
	return f()
}

func (c *Client) fetch(ctx context.Context, method, ref string, body io.Reader, contentType string) (*Page, error) {
	res, err := c.do(ctx, method, ref, body, contentType)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return NewPage(res.Request.URL.String(), res.Body)
}

func (c *Client) do(ctx context.Context, method, ref string, body io.Reader, contentType string) (*http.Response, error) {
	u, err := c.base.Parse(ref)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("manaba: %s %s: %s", method, u, res.Status)
	}
	return res, nil
}

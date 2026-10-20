package manaba

import (
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Page struct {
	URL *url.URL
	Doc *goquery.Document
}

func NewPage(rawURL string, r io.Reader) (*Page, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	return &Page{URL: u, Doc: doc}, nil
}

func (p *Page) Abs(ref string) string {
	u, err := p.URL.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}

func (p *Page) URLWithoutQuery() string {
	u := *p.URL
	u.RawQuery = ""
	return u.String()
}

func parseWith[T any](parse func(*Page) (T, error)) func(*Page, error) (T, error) {
	return func(p *Page, err error) (T, error) {
		if err != nil {
			var zero T
			return zero, err
		}
		return parse(p)
	}
}

func text(s *goquery.Selection) string {
	return strings.Join(strings.Fields(s.Text()), " ")
}

func attr(name string) func(int, *goquery.Selection) string {
	return func(_ int, s *goquery.Selection) string { return s.AttrOr(name, "") }
}

func findOrSelf(s *goquery.Selection, selector string) *goquery.Selection {
	return s.Find(selector).AddSelection(s.Filter(selector))
}

func submatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

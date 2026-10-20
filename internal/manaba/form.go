package manaba

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"maps"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Form struct {
	Action    string
	Multipart bool
	Values    url.Values
}

type Button struct {
	Name  string
	Value string
}

func (b Button) selector() string {
	return fmt.Sprintf(`input[name=%q]`, b.Name)
}

func findForm(p *Page, has string) (Form, error) {
	sel := p.Doc.Find("form").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return s.Find(has).Length() > 0
	}).First()
	if sel.Length() == 0 {
		return Form{}, fmt.Errorf("manaba: form with %s not found on %s", has, p.URL)
	}

	return Form{
		Action:    p.Abs(sel.AttrOr("action", "")),
		Multipart: sel.AttrOr("enctype", "") == "multipart/form-data",
		Values:    hiddenValues(sel),
	}, nil
}

func hiddenValues(form *goquery.Selection) url.Values {
	v := url.Values{}
	for _, in := range form.Find(`input[type="hidden"][name]`).EachIter() {
		v.Add(in.AttrOr("name", ""), in.AttrOr("value", ""))
	}
	return v
}

func (f Form) With(values url.Values) Form {
	merged := maps.Clone(f.Values)
	maps.Copy(merged, values)
	return Form{Action: f.Action, Multipart: f.Multipart, Values: merged}
}

func (f Form) Press(b Button) Form {
	return f.With(url.Values{b.Name: {b.Value}})
}

func (f Form) encode() (io.Reader, string, error) {
	if !f.Multipart {
		return strings.NewReader(f.Values.Encode()), "application/x-www-form-urlencoded", nil
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range pairs(f.Values) {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func pairs(v url.Values) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, vs := range v {
			for _, s := range vs {
				if !yield(k, s) {
					return
				}
			}
		}
	}
}

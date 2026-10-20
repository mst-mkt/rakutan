package manaba

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Jar struct {
	*cookiejar.Jar
	base *url.URL
}

func NewJar(cookies []Cookie) *Jar {
	inner, _ := cookiejar.New(nil)
	base, _ := url.Parse(BaseURL)

	restored := make([]*http.Cookie, len(cookies))
	for i, c := range cookies {
		restored[i] = &http.Cookie{Name: c.Name, Value: c.Value, Path: "/"}
	}
	inner.SetCookies(base, restored)

	return &Jar{Jar: inner, base: base}
}

func (j *Jar) Export() []Cookie {
	cookies := j.Cookies(j.base)
	exported := make([]Cookie, len(cookies))
	for i, c := range cookies {
		exported[i] = Cookie{Name: c.Name, Value: c.Value}
	}
	return exported
}

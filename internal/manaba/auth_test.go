package manaba

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

const loginPage = `<form action="login" method="POST">
<input type="text" name="userid"><input type="password" name="password">
<input type="submit" name="login" value="ログイン">
<input type="hidden" name="manaba-form" value="1"><input type="hidden" name="SessionValue1" value="token"><input type="hidden" name="SessionValue" value="@1">
</form>`

func fakeManaba(t *testing.T, password string) (*Client, *url.Values) {
	t.Helper()
	var posted url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("GET /local/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sessionid", Value: "anonymous", Path: "/"})
		fmt.Fprint(w, loginPage)
	})
	mux.HandleFunc("POST /local/login", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		posted = r.PostForm
		if r.PostForm.Get("password") != password {
			fmt.Fprint(w, loginPage)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "sessionid", Value: "authenticated", Path: "/"})
		fmt.Fprint(w, `<script>location.href = "home";</script>`)
	})
	mux.HandleFunc("GET /local/home", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sessionid"); err != nil || c.Value != "authenticated" {
			fmt.Fprint(w, loginPage)
			return
		}
		fmt.Fprint(w, `<a href="logout">ログアウト</a>`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	jar, _ := cookiejar.New(nil)
	c := New(&http.Client{Jar: jar}, nil)
	c.base, _ = url.Parse(server.URL + "/local/")
	return c, &posted
}

func TestLoginSucceedsWithoutLogoutLinkInResponse(t *testing.T) {
	c, posted := fakeManaba(t, "correct")
	if err := c.Login(t.Context(), "s1F123", "correct"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	assertEqual(t, *posted, url.Values{
		"userid":        {"s1F123"},
		"password":      {"correct"},
		"login":         {"ログイン"},
		"manaba-form":   {"1"},
		"SessionValue1": {"token"},
		"SessionValue":  {"@1"},
	})
}

func TestLoginFailsWithWrongPassword(t *testing.T) {
	c, _ := fakeManaba(t, "correct")
	if err := c.Login(t.Context(), "s1F123", "wrong"); !errors.Is(err, ErrLoginFailed) {
		t.Errorf("err = %v, want ErrLoginFailed", err)
	}
}

package trace

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) (*Trace, *http.Client, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/sheet", http.StatusFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
		w.Write(append([]byte("<p>echo</p>"), body...))
	}))
	t.Cleanup(server.Close)

	tr, err := Open(filepath.Join(t.TempDir(), "debug"), "test", 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.Close() })
	return tr, &http.Client{Transport: tr.Transport(http.DefaultTransport)}, server.URL
}

func readLog(t *testing.T, tr *Trace) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(tr.dir, "rakutan.log"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRedactsSecrets(t *testing.T) {
	tr, client, base := setup(t)
	form := url.Values{"userid": {"s1F123"}, "password": {"hunter2"}, "SessionValue1": {"token"}}
	res, err := client.PostForm(base+"/login", form)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if !strings.Contains(string(body), "hunter2") {
		t.Fatalf("server did not receive the original body: %q", body)
	}
	log := readLog(t, tr)
	for _, secret := range []string{"hunter2", "token"} {
		if strings.Contains(log, secret) {
			t.Errorf("log contains %q:\n%s", secret, log)
		}
	}
	if !strings.Contains(log, `userid=\"s1F123\"`) || !strings.Contains(log, `password=\"[redacted]\"`) {
		t.Errorf("form not described:\n%s", log)
	}
}

func TestDumpsResponsesAndKeepsBody(t *testing.T) {
	tr, client, base := setup(t)
	res, err := client.Get(base + "/start")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "<p>echo</p>" {
		t.Errorf("body = %q", body)
	}

	log := readLog(t, tr)
	for _, want := range []string{"status=302", "location=/sheet", "status=200", "body=002-GET-sheet.html"} {
		if !strings.Contains(log, want) {
			t.Errorf("log lacks %q:\n%s", want, log)
		}
	}
	dumped, err := os.ReadFile(filepath.Join(tr.dir, "002-GET-sheet.html"))
	if err != nil || string(dumped) != "<p>echo</p>" {
		t.Errorf("dump = %q, %v", dumped, err)
	}
}

func TestDescribesMultipartForm(t *testing.T) {
	tr, client, base := setup(t)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("qid1", "90")
	w.WriteField("SessionValue1", "token")
	w.Close()

	res, err := client.Post(base+"/answer", w.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	log := readLog(t, tr)
	if !strings.Contains(log, `qid1=\"90\"`) || strings.Contains(log, "token") {
		t.Errorf("multipart form not described safely:\n%s", log)
	}
}

func TestOpenKeepsRecentRuns(t *testing.T) {
	root := filepath.Join(t.TempDir(), "debug")
	for _, old := range []string{"2026-10-01_10-00-00_take", "2026-10-02_10-00-00_take", "2026-10-03_10-00-00_log"} {
		if err := os.MkdirAll(filepath.Join(root, old), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	tr, err := Open(root, "take", 3)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	run := filepath.Base(tr.dir)
	want := []string{"2026-10-02_10-00-00_take", "2026-10-03_10-00-00_log", run, "latest"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("entries = %v, want %v", names, want)
	}
	if !strings.HasSuffix(run, "_take") {
		t.Errorf("run dir = %s, want *_take", run)
	}
	if target, err := os.Readlink(filepath.Join(root, "latest")); err != nil || target != run {
		t.Errorf("latest -> %q, %v, want %q", target, err, run)
	}
	if info, err := os.Stat(tr.dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("run dir mode = %v, %v", info.Mode(), err)
	}
}

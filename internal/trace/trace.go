package trace

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

var redactedFields = []string{"password", "SessionValue1"}

type Trace struct {
	dir    string
	file   *os.File
	logger *slog.Logger
	seq    atomic.Int64
}

const latest = "latest"

func Open(root, name string, keep int) (*Trace, error) {
	run := time.Now().Format("2006-01-02_15-04-05") + "_" + name
	dir := filepath.Join(root, run)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := prune(root, keep); err != nil {
		return nil, err
	}
	link := filepath.Join(root, latest)
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Symlink(run, link); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filepath.Join(dir, "rakutan.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return &Trace{dir: dir, file: f, logger: logger}, nil
}

func prune(root string, keep int) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	var runs []string
	for _, e := range entries {
		if e.IsDir() {
			runs = append(runs, e.Name())
		}
	}
	slices.Sort(runs)
	for _, run := range runs[:max(0, len(runs)-keep)] {
		if err := os.RemoveAll(filepath.Join(root, run)); err != nil {
			return err
		}
	}
	return nil
}

func (t *Trace) Logger() *slog.Logger {
	return t.logger
}

func (t *Trace) Close() error {
	return t.file.Close()
}

func (t *Trace) Transport(next http.RoundTripper) http.RoundTripper {
	return roundTripper{trace: t, next: next}
}

type roundTripper struct {
	trace *Trace
	next  http.RoundTripper
}

func (rt roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	n := rt.trace.seq.Add(1)
	form, err := readForm(req)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	res, err := rt.next.RoundTrip(req)
	attrs := []any{"n", n, "method", req.Method, "url", req.URL.String(), "elapsed", time.Since(start).Round(time.Millisecond)}
	if form != "" {
		attrs = append(attrs, "form", form)
	}
	if err != nil {
		rt.trace.logger.Error("http", append(attrs, "err", err)...)
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	res.Body = io.NopCloser(bytes.NewReader(body))

	attrs = append(attrs, "status", res.StatusCode)
	if loc := res.Header.Get("Location"); loc != "" {
		attrs = append(attrs, "location", loc)
	}
	if len(body) > 0 {
		attrs = append(attrs, "body", rt.trace.dump(n, req, res, body))
	}
	rt.trace.logger.Debug("http", attrs...)
	return res, nil
}

func (t *Trace) dump(n int64, req *http.Request, res *http.Response, body []byte) string {
	base, ext := baseName(req.URL), extension(res.Header.Get("Content-Type"), body)
	if strings.HasSuffix(base, ext) {
		ext = ""
	}
	name := fmt.Sprintf("%03d-%s-%s%s", n, req.Method, base, ext)
	if err := os.WriteFile(filepath.Join(t.dir, name), body, 0o600); err != nil {
		t.logger.Warn("dump", "file", name, "err", err)
	}
	return name
}

func readForm(req *http.Request) (string, error) {
	if req.Body == nil {
		return "", nil
	}
	data, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(data))

	values, err := parseForm(req.Header.Get("Content-Type"), data)
	if err != nil {
		return fmt.Sprintf("(unparsed: %v)", err), nil
	}
	return describe(values), nil
}

func parseForm(contentType string, data []byte) (url.Values, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case "application/x-www-form-urlencoded":
		return url.ParseQuery(string(data))
	case "multipart/form-data":
		form, err := multipart.NewReader(bytes.NewReader(data), params["boundary"]).ReadForm(1 << 20)
		if err != nil {
			return nil, err
		}
		return form.Value, nil
	default:
		return nil, fmt.Errorf("unsupported content type %s", mediaType)
	}
}

func describe(values url.Values) string {
	var pairs []string
	for _, k := range slices.Sorted(maps.Keys(values)) {
		for _, v := range values[k] {
			if slices.Contains(redactedFields, k) {
				v = "[redacted]"
			}
			pairs = append(pairs, fmt.Sprintf("%s=%q", k, v))
		}
	}
	return strings.Join(pairs, " ")
}

func baseName(u *url.URL) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, path.Base(u.Path))
	if r := []rune(name); len(r) > 60 {
		name = string(r[:60])
	}
	return name
}

func extension(contentType string, body []byte) string {
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch {
	case mediaType == "text/html":
		return ".html"
	case strings.HasPrefix(mediaType, "image/"):
		return "." + strings.TrimPrefix(mediaType, "image/")
	default:
		return ".bin"
	}
}

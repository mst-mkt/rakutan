package archive

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

//go:embed archive.md.tmpl
var tmplText string

var tmpl = template.Must(template.New("archive").Funcs(template.FuncMap{"join": strings.Join}).Parse(tmplText))

type Downloader interface {
	Download(ctx context.Context, ref string) ([]byte, error)
}

type image struct {
	src string
	rel string
}

func Save(ctx context.Context, dl Downloader, root string, r *manaba.Review) (string, error) {
	dir := filepath.Join(root, sanitize(r.Course))

	local := map[string]string{}
	for img := range images(r) {
		data, err := dl.Download(ctx, img.src)
		if err != nil {
			return "", err
		}
		if err := writeFile(filepath.Join(dir, img.rel), data); err != nil {
			return "", err
		}
		local[img.src] = img.rel
	}

	var md bytes.Buffer
	if err := Render(&md, r, local); err != nil {
		return "", err
	}

	file := filepath.Join(dir, sanitize(r.Title)+".md")
	return file, writeFile(file, md.Bytes())
}

func Render(w io.Writer, r *manaba.Review, images map[string]string) error {
	return tmpl.Execute(w, struct {
		*manaba.Review
		Images map[string]string
	}{r, images})
}

func images(r *manaba.Review) iter.Seq[image] {
	return func(yield func(image) bool) {
		for _, q := range r.Questions {
			for i, src := range q.Images {
				rel := fmt.Sprintf("images/%s-%s-%d%s", sanitize(r.Title), q.ID, i+1, extension(src))
				if !yield(image{src: src, rel: rel}) {
					return
				}
			}
		}
	}
}

func writeFile(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, data, 0o644)
}

func extension(src string) string {
	p, _, _ := strings.Cut(src, "?")
	return path.Ext(p)
}

func sanitize(name string) string {
	return strings.NewReplacer("/", "／", "\\", "＼", "\x00", "").Replace(name)
}

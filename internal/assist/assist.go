package assist

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

//go:embed prompt.md
var instructions string

var command = "claude"

const schema = `{
  "type": "object",
  "properties": {
    "answers": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "values": {"type": "array", "items": {"type": "string"}}
        },
        "required": ["id", "values"]
      }
    }
  },
  "required": ["answers"]
}`

type Downloader interface {
	Download(ctx context.Context, ref string) ([]byte, error)
}

type question struct {
	ID      string          `json:"id"`
	Kind    manaba.Kind     `json:"kind"`
	Text    string          `json:"text"`
	Options []manaba.Option `json:"options,omitempty"`
	Images  []string        `json:"images,omitempty"`
}

type output struct {
	IsError          bool   `json:"is_error"`
	Result           string `json:"result"`
	StructuredOutput struct {
		Answers []struct {
			ID     string   `json:"id"`
			Values []string `json:"values"`
		} `json:"answers"`
	} `json:"structured_output"`
}

func Suggest(ctx context.Context, dl Downloader, sheet *manaba.Sheet, references string) (manaba.Answers, error) {
	work, err := os.MkdirTemp("", "rakutan-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)

	questions, err := prepare(ctx, dl, sheet.Questions, work)
	if err != nil {
		return nil, err
	}
	prompt, err := json.MarshalIndent(map[string]any{"title": sheet.Title, "questions": questions}, "", "  ")
	if err != nil {
		return nil, err
	}

	dirs := []string{work}
	if info, err := os.Stat(references); err == nil && info.IsDir() {
		dirs = append(dirs, references)
	}
	out, err := run(ctx, work, string(prompt), dirs)
	if err != nil {
		return nil, err
	}
	return answers(sheet.Questions, out), nil
}

func prepare(ctx context.Context, dl Downloader, qs []manaba.Question, dir string) ([]question, error) {
	prepared := make([]question, len(qs))
	for i, q := range qs {
		images := make([]string, len(q.Images))
		for j, src := range q.Images {
			data, err := dl.Download(ctx, src)
			if err != nil {
				return nil, err
			}
			images[j] = filepath.Join(dir, fmt.Sprintf("%s-%d%s", q.ID, j+1, extension(src)))
			if err := os.WriteFile(images[j], data, 0o600); err != nil {
				return nil, err
			}
		}
		prepared[i] = question{ID: q.ID, Kind: q.Kind, Text: q.Text, Options: q.Options, Images: images}
	}
	return prepared, nil
}

func run(ctx context.Context, work, prompt string, dirs []string) (output, error) {
	args := []string{
		"-p",
		"--output-format", "json",
		"--json-schema", schema,
		"--append-system-prompt", instructions,
		"--tools", "Read,Grep,Glob",
		"--allowedTools", "Read", "Grep", "Glob",
		"--no-session-persistence",
		"--add-dir",
	}
	args = append(args, dirs...)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = work
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	start := time.Now()
	err := cmd.Run()
	slog.Debug("assist", "args", args, "prompt", prompt, "elapsed", time.Since(start).Round(time.Millisecond), "stdout", stdout.String(), "stderr", stderr.String(), "err", err)
	if err != nil {
		return output{}, fmt.Errorf("%s: %w: %s", command, err, strings.TrimSpace(stderr.String()))
	}

	var out output
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return output{}, fmt.Errorf("%s の出力を読めません: %w", command, err)
	}
	if out.IsError {
		return output{}, errors.New(out.Result)
	}
	return out, nil
}

func answers(qs []manaba.Question, out output) manaba.Answers {
	byID := make(map[string][]string, len(out.StructuredOutput.Answers))
	for _, a := range out.StructuredOutput.Answers {
		byID[a.ID] = a.Values
	}

	ans := make(manaba.Answers, len(qs))
	for _, q := range qs {
		ans[q.ID] = valid(q, byID[q.ID])
	}
	return ans
}

func valid(q manaba.Question, values []string) []string {
	if q.Kind == manaba.KindText || q.Kind == manaba.KindTextarea {
		if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
			return nil
		}
		return values[:1]
	}

	var picked []string
	for _, v := range values {
		if slices.ContainsFunc(q.Options, func(o manaba.Option) bool { return o.Value == v }) && !slices.Contains(picked, v) {
			picked = append(picked, v)
		}
	}
	if q.Kind != manaba.KindCheckbox {
		return picked[:min(len(picked), 1)]
	}
	return picked
}

func extension(src string) string {
	p, _, _ := strings.Cut(src, "?")
	return path.Ext(p)
}

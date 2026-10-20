package assist

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/mst-mkt/rakutan/internal/manaba"
)

type fakeDownloader map[string][]byte

func (f fakeDownloader) Download(_ context.Context, ref string) ([]byte, error) {
	return f[ref], nil
}

var sheet = &manaba.Sheet{
	Title: "第1回",
	Questions: []manaba.Question{
		{ID: "qid1", Kind: manaba.KindText, Text: "Q1", Images: []string{"https://example.com/fig.png?view=inline"}},
		{ID: "qid2", Kind: manaba.KindCheckbox, Text: "Q2", Options: []manaba.Option{{Value: "1", Label: "A"}, {Value: "2", Label: "B"}}},
		{ID: "qid3", Kind: manaba.KindSelect, Text: "Q3", Options: []manaba.Option{{Value: "1", Label: "A"}, {Value: "2", Label: "B"}}},
		{ID: "qid4", Kind: manaba.KindRadio, Text: "Q4", Options: []manaba.Option{{Value: "1", Label: "A"}}},
	},
}

func fakeClaude(t *testing.T, stdout string) (argsFile, stdinFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile, stdinFile = filepath.Join(dir, "args"), filepath.Join(dir, "stdin")
	script := "#!/bin/sh\n" +
		`printf '%s\n' "$@" > ` + argsFile + "\n" +
		"cat > " + stdinFile + "\n" +
		"ls \"$PWD\" >> " + stdinFile + "\n" +
		"cat <<'EOF'\n" + stdout + "\nEOF\n"
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	previous := command
	command = bin
	t.Cleanup(func() { command = previous })
	return argsFile, stdinFile
}

func TestSuggest(t *testing.T) {
	argsFile, stdinFile := fakeClaude(t, `{"is_error": false, "structured_output": {"answers": [
		{"id": "qid1", "values": ["1200", "extra"]},
		{"id": "qid2", "values": ["2", "9", "2", "1"]},
		{"id": "qid3", "values": ["2", "1"]},
		{"id": "qid4", "values": ["unknown"]},
		{"id": "qid9", "values": ["1"]}
	]}}`)
	references := t.TempDir()

	got, err := Suggest(t.Context(), fakeDownloader{sheet.Questions[0].Images[0]: []byte("png")}, sheet, references)
	if err != nil {
		t.Fatal(err)
	}
	want := manaba.Answers{"qid1": {"1200"}, "qid2": {"2", "1"}, "qid3": {"2"}, "qid4": nil}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("answers mismatch (-want +got):\n%s", diff)
	}

	args, _ := os.ReadFile(argsFile)
	for _, want := range []string{"-p\n", "--tools\nRead,Grep,Glob\n", "--no-session-persistence\n", references + "\n"} {
		if !strings.Contains(string(args), want) {
			t.Errorf("args lack %q:\n%s", want, args)
		}
	}
	stdin, _ := os.ReadFile(stdinFile)
	for _, want := range []string{`"id": "qid1"`, `"title": "第1回"`, "qid1-1.png"} {
		if !strings.Contains(string(stdin), want) {
			t.Errorf("prompt or work dir lacks %q:\n%s", want, stdin)
		}
	}
}

func TestSuggestSkipsMissingReferences(t *testing.T) {
	argsFile, _ := fakeClaude(t, `{"is_error": false, "structured_output": {"answers": []}}`)
	if _, err := Suggest(t.Context(), fakeDownloader{}, &manaba.Sheet{}, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(argsFile)
	if strings.Contains(string(args), "missing") {
		t.Errorf("missing reference dir passed:\n%s", args)
	}
}

func TestSuggestReportsError(t *testing.T) {
	fakeClaude(t, `{"is_error": true, "result": "usage limit reached"}`)
	_, err := Suggest(t.Context(), fakeDownloader{}, &manaba.Sheet{}, "")
	if err == nil || !strings.Contains(err.Error(), "usage limit reached") {
		t.Errorf("err = %v", err)
	}
}

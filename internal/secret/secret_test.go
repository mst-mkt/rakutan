package secret

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

type pair struct {
	Name  string
	Value string
}

func TestKeyring(t *testing.T) {
	keyring.MockInit()
	fallback := filepath.Join(t.TempDir(), "secret.json")
	s := WithFileFallback[pair]("test", fallback)

	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load before Save: err = %v, want ErrNotFound", err)
	}
	if err := s.Save(pair{"a", "1"}); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Load(); err != nil || got != (pair{"a", "1"}) {
		t.Fatalf("Load = %v, %v", got, err)
	}
	if _, err := os.Stat(fallback); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("fallback file written while keyring works: %v", err)
	}

	if err := s.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Errorf("Load after Delete: err = %v, want ErrNotFound", err)
	}
}

func TestFallbackWhenKeyringUnavailable(t *testing.T) {
	keyring.MockInitWithError(errors.New("no secret service"))
	fallback := filepath.Join(t.TempDir(), "dir", "secret.json")
	s := WithFileFallback[pair]("test", fallback)

	if err := s.Save(pair{"b", "2"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(fallback); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("fallback file: %v, %v", info, err)
	}
	if got, err := s.Load(); err != nil || got != (pair{"b", "2"}) {
		t.Fatalf("Load = %v, %v", got, err)
	}

	if err := s.Delete(); err == nil {
		t.Error("Delete should report the keyring error")
	}
	if _, err := os.Stat(fallback); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("fallback file remains after Delete: %v", err)
	}
}

func TestNoFallbackNeverWritesFile(t *testing.T) {
	keyring.MockInitWithError(errors.New("no secret service"))
	if err := New[pair]("test").Save(pair{"c", "3"}); err == nil {
		t.Error("Save without fallback should fail when keyring is unavailable")
	}
}

package secret

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const service = "rakutan"

var ErrNotFound = errors.New("secret: not found")

type Secret[T any] struct {
	key      string
	fallback string
}

func New[T any](key string) Secret[T] {
	return Secret[T]{key: key}
}

func WithFileFallback[T any](key, path string) Secret[T] {
	return Secret[T]{key: key, fallback: path}
}

func (s Secret[T]) Load() (T, error) {
	var v T
	data, err := s.read()
	if err != nil {
		return v, err
	}
	return v, json.Unmarshal(data, &v)
}

func (s Secret[T]) Save(v T) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	err = keyring.Set(service, s.key, string(data))
	if err == nil || s.fallback == "" {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.fallback), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.fallback, data, 0o600)
}

func (s Secret[T]) Delete() error {
	return errors.Join(
		ignoreNotFound(keyring.Delete(service, s.key)),
		ignoreNotFound(s.removeFallback()),
	)
}

func (s Secret[T]) read() ([]byte, error) {
	v, err := keyring.Get(service, s.key)
	switch {
	case err == nil:
		return []byte(v), nil
	case errors.Is(err, keyring.ErrNotFound), s.fallback == "":
		return nil, notFound(err)
	}

	data, err := os.ReadFile(s.fallback)
	return data, notFound(err)
}

func (s Secret[T]) removeFallback() error {
	if s.fallback == "" {
		return nil
	}
	return os.Remove(s.fallback)
}

func notFound(err error) error {
	if errors.Is(err, keyring.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	}
	return err
}

func ignoreNotFound(err error) error {
	if errors.Is(notFound(err), ErrNotFound) {
		return nil
	}
	return err
}

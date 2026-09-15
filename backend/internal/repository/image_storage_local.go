package repository

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
)

type LocalImageStorage struct{ directory string }

func NewLocalImageStorage(cfg *config.ImageStorageConfig) (*LocalImageStorage, error) {
	dir, err := filepath.Abs(cfg.LocalDirectory)
	if err != nil || strings.TrimSpace(cfg.LocalDirectory) == "" {
		return nil, errors.New("invalid image directory")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &LocalImageStorage{directory: dir}, nil
}

// Only generated single filenames are used locally, never user-supplied paths.
func (s *LocalImageStorage) filename(key string) (string, error) {
	base := filepath.Base(filepath.FromSlash(key))
	if strings.Contains(key, "..") || strings.Contains(key, `\`) || strings.ContainsAny(base, `\/:`) || !(strings.Contains(key, "/studio/") || strings.HasPrefix(key, "studio/")) {
		return "", errors.New("invalid image object key")
	}
	if _, err := uuid.Parse(strings.TrimSuffix(base, filepath.Ext(base))); err != nil {
		return "", errors.New("invalid image filename")
	}
	return filepath.Join(s.directory, base), nil
}

func (s *LocalImageStorage) Save(ctx context.Context, key, ct string, data []byte) (string, error) {
	_, err := s.Put(ctx, key, ct, data)
	return "", err
}

func (s *LocalImageStorage) Put(ctx context.Context, key, ct string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, err := s.filename(key)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return "", writeErr
	}
	return "", closeErr
}

func (s *LocalImageStorage) Open(ctx context.Context, key, version string) (io.ReadCloser, error) {
	name, err := s.filename(key)
	if err != nil {
		return nil, err
	}
	return os.Open(name)
}

func (s *LocalImageStorage) Delete(ctx context.Context, key, version string) error {
	name, err := s.filename(key)
	if err != nil {
		return err
	}
	err = os.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

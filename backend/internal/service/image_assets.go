package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const ImageRetention = 2 * time.Hour

// ImageBlobStore keeps provider-specific operations out of the retention logic.
// Version IDs ensure versioned buckets delete the actual bytes, not just a marker.
type ImageBlobStore interface {
	Put(context.Context, string, string, []byte) (string, error)
	Open(context.Context, string, string) (io.ReadCloser, error)
	Delete(context.Context, string, string) error
}

type ImageAssetRecord struct {
	ID, TaskID, Key, ContentType, VersionID string
	Config                                  json.RawMessage
	Ready                                   bool
	ExpiresAt                               time.Time
}

type ImageAssetRepository interface {
	Create(context.Context, *ImageAssetRecord) error
	Ready(context.Context, string, string, time.Time) error
	Get(context.Context, string) (*ImageAssetRecord, error)
	Due(context.Context, time.Time, int) ([]*ImageAssetRecord, error)
	Retry(context.Context, string, time.Time) error
	Delete(context.Context, string) error
}

// ImageAssetService persists a deletion journal before writing any bytes. Each
// entry carries its original storage binding (with encrypted credentials), so a
// restart or an admin configuration change cannot strand the previous objects.
type ImageAssetService struct {
	repo      ImageAssetRepository
	factory   ImageStorageFactory
	encryptor SecretEncryptor
	stop      chan struct{}
	done      chan struct{}
	stopOnce  sync.Once
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewImageAssetService(repo ImageAssetRepository, factory ImageStorageFactory, encryptor SecretEncryptor) *ImageAssetService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ImageAssetService{repo: repo, factory: factory, encryptor: encryptor, stop: make(chan struct{}), done: make(chan struct{}), ctx: ctx, cancel: cancel}
}

func (s *ImageAssetService) Start() {
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(s.ctx, 50*time.Second)
			if err := s.Cleanup(ctx, time.Now()); err != nil {
				logger.L().Warn("image_assets.cleanup_failed", zap.Error(err))
			}
			cancel()
			select {
			case <-s.stop:
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *ImageAssetService) Stop() {
	s.stopOnce.Do(func() { s.cancel(); close(s.stop) })
	<-s.done
}

func (s *ImageAssetService) Storage(cfg *config.ImageStorageConfig, raw ImageStorage) (ImageStorage, error) {
	blob, ok := raw.(ImageBlobStore)
	if !ok {
		return nil, errors.New("image storage does not support retention")
	}
	snapshot := *cfg
	if snapshot.Provider == "local" {
		absolute, err := filepath.Abs(snapshot.LocalDirectory)
		if err != nil {
			return nil, err
		}
		snapshot.LocalDirectory = absolute
	}
	if snapshot.SecretAccessKey != "" {
		secret, err := s.encryptor.Encrypt(snapshot.SecretAccessKey)
		if err != nil {
			return nil, err
		}
		snapshot.SecretAccessKey = secret
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return &retainedImageStorage{service: s, blob: blob, config: encoded, prefix: strings.Trim(cfg.Prefix, "/")}, nil
}

type retainedImageStorage struct {
	service *ImageAssetService
	blob    ImageBlobStore
	config  json.RawMessage
	prefix  string
}

func (s *retainedImageStorage) Save(ctx context.Context, key, contentType string, data []byte) (string, error) {
	id := uuid.NewString()
	base := path.Base(key)
	pos := strings.LastIndex(base, "-")
	if pos < 0 || !strings.HasPrefix(base, "imgtask_") {
		return "", errors.New("invalid image task storage key")
	}
	record := &ImageAssetRecord{
		ID: id, TaskID: base[:pos], Key: path.Join(s.prefix, "studio", id+extensionForContentType(contentType)),
		ContentType: contentType, Config: s.config, ExpiresAt: time.Now().Add(ImageRetention),
	}
	if err := s.service.repo.Create(ctx, record); err != nil {
		return "", err
	}
	version, err := s.blob.Put(ctx, record.Key, contentType, data)
	if err != nil {
		// Keep the journal even on ambiguous upload failure, for a later delete.
		return "", err
	}
	if err := s.service.repo.Ready(ctx, id, version, time.Now().Add(ImageRetention)); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if deleteErr := s.blob.Delete(cleanupCtx, record.Key, version); deleteErr == nil {
			_ = s.service.repo.Delete(cleanupCtx, id)
		}
		return "", err
	}
	return "/v1/images/assets/" + id, nil
}

func (s *ImageAssetService) binding(ctx context.Context, record *ImageAssetRecord) (ImageBlobStore, error) {
	var cfg config.ImageStorageConfig
	if err := json.Unmarshal(record.Config, &cfg); err != nil {
		return nil, err
	}
	if cfg.SecretAccessKey != "" {
		secret, err := s.encryptor.Decrypt(cfg.SecretAccessKey)
		if err != nil {
			return nil, err
		}
		cfg.SecretAccessKey = secret
	}
	storage, err := s.factory(ctx, &cfg)
	if err != nil {
		return nil, err
	}
	blob, ok := storage.(ImageBlobStore)
	if !ok {
		return nil, errors.New("image storage does not support retention")
	}
	return blob, nil
}

func (s *ImageAssetService) Get(ctx context.Context, id string) (*ImageAssetRecord, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrImageTaskNotFound
	}
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !record.Ready || !time.Now().Before(record.ExpiresAt) {
		return nil, ErrImageTaskNotFound
	}
	return record, nil
}

func (s *ImageAssetService) Open(ctx context.Context, record *ImageAssetRecord) (io.ReadCloser, error) {
	if !time.Now().Before(record.ExpiresAt) {
		return nil, ErrImageTaskNotFound
	}
	blob, err := s.binding(ctx, record)
	if err != nil {
		return nil, err
	}
	return blob.Open(ctx, record.Key, record.VersionID)
}

// TestStorage uses the same durable journal as generation. A failed cleanup is
// retried by the worker, including when the tested configuration was not saved.
func (s *ImageAssetService) TestStorage(ctx context.Context, cfg *config.ImageStorageConfig, raw ImageStorage) error {
	storage, err := s.Storage(cfg, raw)
	if err != nil {
		return err
	}
	payload := []byte("sub2api image storage check")
	url, err := storage.Save(ctx, "imgtask_storagecheck-0.png", "application/octet-stream", payload)
	if err != nil {
		return fmt.Errorf("storage write check: %w", err)
	}
	id := strings.TrimPrefix(url, "/v1/images/assets/")
	record, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	body, readErr := s.Open(ctx, record)
	if readErr == nil {
		data, err := io.ReadAll(io.LimitReader(body, int64(len(payload)+1)))
		closeErr := body.Close()
		readErr = errors.Join(err, closeErr)
		if readErr == nil && !bytes.Equal(data, payload) {
			readErr = errors.New("storage returned different bytes")
		}
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	blob := raw.(ImageBlobStore) // Storage already validated this interface.
	deleteErr := blob.Delete(cleanupCtx, record.Key, record.VersionID)
	if deleteErr == nil {
		deleteErr = s.repo.Delete(cleanupCtx, id)
	}
	if readErr != nil {
		return fmt.Errorf("storage read check: %w", readErr)
	}
	if deleteErr != nil {
		return fmt.Errorf("storage delete check: %w", deleteErr)
	}
	return nil
}

func (s *ImageAssetService) Cleanup(ctx context.Context, now time.Time) error {
	// Bound each sweep, but drain multiple pages to avoid a 100 images/min ceiling.
	for page := 0; page < 20; page++ {
		records, err := s.repo.Due(ctx, now, 100)
		if err != nil {
			return err
		}
		for _, record := range records {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			blob, deleteErr := s.binding(ctx, record)
			if deleteErr == nil {
				deleteErr = blob.Delete(ctx, record.Key, record.VersionID)
			}
			if deleteErr == nil {
				deleteErr = s.repo.Delete(ctx, record.ID)
			}
			if deleteErr != nil {
				logger.L().Warn("image_assets.delete_failed", zap.String("asset_id", record.ID), zap.Error(deleteErr))
				if err := s.repo.Retry(ctx, record.ID, now.Add(time.Minute)); err != nil {
					return fmt.Errorf("defer image cleanup: %w", err)
				}
			}
		}
		if len(records) < 100 {
			return nil
		}
	}
	return nil
}

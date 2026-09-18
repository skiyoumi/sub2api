//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type assetMemoryRepo struct {
	records   map[string]*ImageAssetRecord
	retry     map[string]time.Time
	createErr error
}

func newAssetMemoryRepo() *assetMemoryRepo {
	return &assetMemoryRepo{records: map[string]*ImageAssetRecord{}, retry: map[string]time.Time{}}
}
func (r *assetMemoryRepo) Create(_ context.Context, a *ImageAssetRecord) error {
	if r.createErr != nil {
		return r.createErr
	}
	copy := *a
	r.records[a.ID] = &copy
	return nil
}
func (r *assetMemoryRepo) Ready(_ context.Context, id, version string, expires time.Time) error {
	r.records[id].Ready = true
	r.records[id].VersionID = version
	r.records[id].ExpiresAt = expires
	return nil
}
func (r *assetMemoryRepo) Get(_ context.Context, id string) (*ImageAssetRecord, error) {
	a, ok := r.records[id]
	if !ok {
		return nil, ErrImageTaskNotFound
	}
	return a, nil
}
func (r *assetMemoryRepo) Due(_ context.Context, now time.Time, limit int) ([]*ImageAssetRecord, error) {
	out := []*ImageAssetRecord{}
	for _, a := range r.records {
		if !a.ExpiresAt.After(now) && !r.retry[a.ID].After(now) {
			out = append(out, a)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}
func (r *assetMemoryRepo) Retry(_ context.Context, id string, at time.Time) error {
	r.retry[id] = at
	return nil
}
func (r *assetMemoryRepo) Delete(_ context.Context, id string) error {
	delete(r.records, id)
	return nil
}

type assetMemoryBlob struct {
	data           map[string][]byte
	deleteErr      error
	puts           int
	deletedVersion string
}

func (b *assetMemoryBlob) Save(context.Context, string, string, []byte) (string, error) {
	panic("use Put")
}
func (b *assetMemoryBlob) Put(_ context.Context, key, ct string, data []byte) (string, error) {
	b.puts++
	b.data[key] = data
	return "version-1", nil
}
func (b *assetMemoryBlob) Open(_ context.Context, key, version string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(b.data[key]))), nil
}
func (b *assetMemoryBlob) Delete(_ context.Context, key, version string) error {
	if b.deleteErr != nil {
		return b.deleteErr
	}
	b.deletedVersion = version
	delete(b.data, key)
	return nil
}

func TestImageAssetsRetentionSurvivesRestartAndConfigurationChange(t *testing.T) {
	ctx := context.Background()
	repo := newAssetMemoryRepo()
	blob := &assetMemoryBlob{data: map[string][]byte{}}
	factory := func(_ context.Context, cfg *config.ImageStorageConfig) (ImageStorage, error) {
		require.Equal(t, "old-bucket", cfg.Bucket)
		require.Equal(t, "old-secret", cfg.SecretAccessKey)
		return blob, nil
	}
	svc := NewImageAssetService(repo, factory, reversibleEncryptor{})
	storage, err := svc.Storage(&config.ImageStorageConfig{Provider: "qiniu", Bucket: "old-bucket", SecretAccessKey: "old-secret", Prefix: "images/"}, blob)
	require.NoError(t, err)
	url, err := storage.Save(ctx, "images/imgtask_123-0.png", "image/png", []byte("image-bytes"))
	require.NoError(t, err)
	id := strings.TrimPrefix(url, "/v1/images/assets/")
	a, err := svc.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "imgtask_123", a.TaskID)
	var snapshot config.ImageStorageConfig
	require.NoError(t, json.Unmarshal(a.Config, &snapshot))
	require.Equal(t, "enc:old-secret", snapshot.SecretAccessKey)
	require.WithinDuration(t, time.Now().Add(2*time.Hour), a.ExpiresAt, time.Second)
	body, err := svc.Open(ctx, a)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	body.Close()
	require.NoError(t, err)
	require.Equal(t, "image-bytes", string(data))
	// A newly created service reads the persisted original binding, not new admin settings.
	restarted := NewImageAssetService(repo, factory, reversibleEncryptor{})
	require.NoError(t, restarted.Cleanup(ctx, a.ExpiresAt.Add(-time.Second)))
	require.Len(t, blob.data, 1)
	a.ExpiresAt = time.Now().Add(-time.Second)
	_, err = restarted.Get(ctx, id)
	require.ErrorIs(t, err, ErrImageTaskNotFound)
	blob.deleteErr = errors.New("temporary cloud outage")
	require.NoError(t, restarted.Cleanup(ctx, time.Now()))
	require.Len(t, repo.records, 1)
	require.Len(t, blob.data, 1)
	blob.deleteErr = nil
	require.NoError(t, restarted.Cleanup(ctx, time.Now().Add(2*time.Minute)))
	require.Empty(t, repo.records)
	require.Empty(t, blob.data)
	require.Equal(t, "version-1", blob.deletedVersion)
}

func TestImageAssetsJournalMustExistBeforeUpload(t *testing.T) {
	repo := newAssetMemoryRepo()
	repo.createErr = errors.New("database unavailable")
	blob := &assetMemoryBlob{data: map[string][]byte{}}
	svc := NewImageAssetService(repo, nil, reversibleEncryptor{})
	storage, err := svc.Storage(&config.ImageStorageConfig{Provider: "local", LocalDirectory: t.TempDir(), Prefix: "images/"}, blob)
	require.NoError(t, err)
	_, err = storage.Save(context.Background(), "images/imgtask_123-0.png", "image/png", []byte("bytes"))
	require.Error(t, err)
	require.Zero(t, blob.puts)
}

func TestImageAssetsStorageCheckCleansProbeAndRetainsFailedDeletion(t *testing.T) {
	repo := newAssetMemoryRepo()
	blob := &assetMemoryBlob{data: map[string][]byte{}}
	svc := NewImageAssetService(repo, func(context.Context, *config.ImageStorageConfig) (ImageStorage, error) { return blob, nil }, reversibleEncryptor{})
	cfg := &config.ImageStorageConfig{Provider: "local", LocalDirectory: t.TempDir()}
	require.NoError(t, svc.TestStorage(context.Background(), cfg, blob))
	require.Empty(t, blob.data)
	require.Empty(t, repo.records)
	blob.deleteErr = errors.New("delete permission denied")
	require.ErrorContains(t, svc.TestStorage(context.Background(), cfg, blob), "storage delete check")
	require.Len(t, repo.records, 1, "unsaved test configuration must still be available for cleanup retries")
	blob.deleteErr = nil
	require.NoError(t, svc.Cleanup(context.Background(), time.Now().Add(ImageRetention+time.Minute)))
	require.Empty(t, blob.data)
	require.Empty(t, repo.records)
}

func TestImageTaskCapturesStorageBeforeAdminDisablesIt(t *testing.T) {
	store := &imageTaskMemoryStore{}
	blob := &recordingStorage{}
	uploader := NewImageResultUploader(blob, "original/", 1024, nil)
	current := uploader
	svc := NewImageTaskServiceWithResolver(store, func() (*ImageResultUploader, bool) { return current, current != nil }, ImageRetention, time.Minute)
	task, err := svc.Create(context.Background(), ImageTaskOwner{UserID: 1, APIKeyID: 2})
	require.NoError(t, err)
	current = nil
	require.NoError(t, svc.Complete(context.Background(), task.ID, 200, json.RawMessage(`{"data":[{"b64_json":"aW1hZ2U="}]}`)))
	require.Len(t, blob.saved, 1)
	require.True(t, strings.HasPrefix(blob.saved[0], "original/"))
	_, bound := svc.bindings.Load(task.ID)
	require.False(t, bound)
}

func TestImageAssetsRequireTaskOwnershipBeforeReadingBytes(t *testing.T) {
	ctx := context.Background()
	repo := newAssetMemoryRepo()
	blob := &assetMemoryBlob{data: map[string][]byte{}}
	svc := NewImageAssetService(repo, func(context.Context, *config.ImageStorageConfig) (ImageStorage, error) { return blob, nil }, reversibleEncryptor{})
	store := &imageTaskMemoryStore{}
	tasks := NewImageTaskServiceWithOptions(store, ImageRetention, time.Minute)
	tasks.assets = svc
	owner := ImageTaskOwner{UserID: 1, APIKeyID: 2}
	task, err := tasks.Create(ctx, owner)
	require.NoError(t, err)
	storage, err := svc.Storage(&config.ImageStorageConfig{Prefix: "images/"}, blob)
	require.NoError(t, err)
	url, err := storage.Save(ctx, "images/"+task.ID+"-0.png", "image/png", []byte("private"))
	require.NoError(t, err)
	id := strings.TrimPrefix(url, "/v1/images/assets/")
	_, _, err = tasks.ReadAsset(ctx, ImageTaskOwner{UserID: 1, APIKeyID: 3}, id)
	require.ErrorIs(t, err, ErrImageTaskNotFound)
	body, ct, err := tasks.ReadAsset(ctx, owner, id)
	require.NoError(t, err)
	body.Close()
	require.Equal(t, "image/png", ct)
	repo.records[id].ExpiresAt = time.Now().Add(-time.Second)
	_, _, err = tasks.ReadAsset(ctx, owner, id)
	require.ErrorIs(t, err, ErrImageTaskNotFound)
}

func TestImagePreviewAssetsShareTaskOwnershipAndRetention(t *testing.T) {
	ctx := context.Background()
	repo := newAssetMemoryRepo()
	blob := &assetMemoryBlob{data: map[string][]byte{}}
	assets := NewImageAssetService(repo, func(context.Context, *config.ImageStorageConfig) (ImageStorage, error) { return blob, nil }, reversibleEncryptor{})
	tasks := NewImageTaskServiceWithOptions(&imageTaskMemoryStore{}, ImageRetention, time.Minute)
	tasks.assets = assets
	owner := ImageTaskOwner{UserID: 1, APIKeyID: 2}
	task, err := tasks.Create(ctx, owner)
	require.NoError(t, err)
	storage, err := assets.Storage(&config.ImageStorageConfig{Prefix: "images/"}, blob)
	require.NoError(t, err)
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	original := noisyImagePNG(t, 256, 256)
	input, err := json.Marshal(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(original)}}})
	require.NoError(t, err)
	result, err := uploader.Rewrite(ctx, task.ID, input)
	require.NoError(t, err)
	require.Len(t, repo.records, 2)
	for _, field := range []string{"url", "preview_url"} {
		url := gjson.GetBytes(result, "data.0."+field).String()
		require.True(t, strings.HasPrefix(url, "/v1/images/assets/"))
		id := strings.TrimPrefix(url, "/v1/images/assets/")
		require.Equal(t, task.ID, repo.records[id].TaskID)
		require.WithinDuration(t, time.Now().Add(ImageRetention), repo.records[id].ExpiresAt, time.Second)
		_, _, err := tasks.ReadAsset(ctx, ImageTaskOwner{UserID: 1, APIKeyID: 3}, id)
		require.ErrorIs(t, err, ErrImageTaskNotFound)
		body, ct, err := tasks.ReadAsset(ctx, owner, id)
		require.NoError(t, err)
		data, err := io.ReadAll(body)
		require.NoError(t, body.Close())
		require.NoError(t, err)
		if field == "url" {
			require.Equal(t, original, data)
			require.Equal(t, "image/png", ct)
		} else {
			require.Less(t, len(data), len(original))
			require.Equal(t, "image/jpeg", ct)
		}
		repo.records[id].ExpiresAt = time.Now().Add(-time.Second)
		_, _, err = tasks.ReadAsset(ctx, owner, id)
		require.ErrorIs(t, err, ErrImageTaskNotFound)
	}
	require.NoError(t, assets.Cleanup(ctx, time.Now()))
	require.Empty(t, blob.data)
	require.Empty(t, repo.records)
}

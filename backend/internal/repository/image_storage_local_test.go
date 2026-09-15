package repository

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLocalImageStorageWritesReadsAndDeletesOnlyManagedFiles(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalImageStorage(&config.ImageStorageConfig{LocalDirectory: dir})
	require.NoError(t, err)
	id := uuid.NewString()
	key := "images/studio/" + id + ".png"
	ctx := context.Background()
	_, err = storage.Put(ctx, key, "image/png", []byte("test-image"))
	require.NoError(t, err)
	body, err := storage.Open(ctx, key, "")
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	body.Close()
	require.NoError(t, err)
	require.Equal(t, "test-image", string(data))
	_, err = storage.Put(ctx, key, "image/png", []byte("overwrite"))
	require.Error(t, err)
	for _, bad := range []string{"../secret", "images/studio/../../outside.png", "images/studio/backup.sql", `images/studio/..\secret`} {
		require.Error(t, storage.Delete(ctx, bad, ""))
	}
	require.NoError(t, storage.Delete(ctx, key, ""))
	require.NoError(t, storage.Delete(ctx, key, ""))
	_, err = os.Stat(filepath.Join(dir, id+".png"))
	require.True(t, os.IsNotExist(err))
}

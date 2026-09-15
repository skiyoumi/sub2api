package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestImageTaskStoreRoundTripAndTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewImageTaskStore(rdb)
	task := &service.ImageTaskRecord{
		ID:        "imgtask_123",
		UserID:    7,
		APIKeyID:  9,
		Status:    service.ImageTaskStatusProcessing,
		CreatedAt: 100,
		ExpiresAt: 200,
	}

	require.NoError(t, store.Save(context.Background(), task, 24*time.Hour))
	got, err := store.Get(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, task, got)
	require.Equal(t, 24*time.Hour, mr.TTL(imageTaskKey(task.ID)))
}

func TestImageTaskStoreMissing(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewImageTaskStore(rdb)

	_, err := store.Get(context.Background(), "imgtask_missing")
	require.ErrorIs(t, err, service.ErrImageTaskNotFound)
}

func TestImageTaskHistoryRetainsPromptAndExpiresWithTask(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewImageTaskStore(rdb).(*imageTaskStore)
	svc := service.NewImageTaskServiceWithUploader(store, nil, service.ImageRetention, time.Minute)
	ctx := context.Background()
	owner := service.ImageTaskOwner{UserID: 7, APIKeyID: 9}
	input := &service.ImageTaskRequest{Prompt: "原始提示词\n保留换行", Model: "gpt-image-2", Size: "1234x987", Quality: "high", N: 2}
	task, err := svc.CreateWithRequest(ctx, owner, input)
	require.NoError(t, err)
	input.Prompt = "changed after submission"
	rows, err := svc.List(ctx, owner)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, service.ImageTaskStatusProcessing, rows[0].Status)
	require.Equal(t, "原始提示词\n保留换行", rows[0].Request.Prompt)

	require.NoError(t, svc.Complete(ctx, task.ID, 200, []byte(`{"data":[{"url":"/v1/images/assets/example"}]}`)))
	rows, err = svc.List(ctx, owner)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, service.ImageTaskStatusCompleted, rows[0].Status)
	require.Equal(t, "1234x987", rows[0].Request.Size)
	require.Equal(t, "原始提示词\n保留换行", rows[0].Request.Prompt)
	for _, other := range []service.ImageTaskOwner{{UserID: 7, APIKeyID: 10}, {UserID: 8, APIKeyID: 9}} {
		otherRows, err := svc.List(ctx, other)
		require.NoError(t, err)
		require.Empty(t, otherRows)
		_, err = svc.Get(ctx, other, task.ID)
		require.ErrorIs(t, err, service.ErrImageTaskNotFound)
	}
	mr.FastForward(service.ImageRetention + time.Second)
	_, err = store.Get(ctx, task.ID)
	require.ErrorIs(t, err, service.ErrImageTaskNotFound)
	rows, err = svc.List(ctx, owner)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.False(t, mr.Exists(imageTaskKey(task.ID)))
	require.False(t, mr.Exists("image_task_index:7:9"))
}

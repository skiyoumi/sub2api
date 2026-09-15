package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const imageTaskKeyPrefix = "image_task:"

type imageTaskStore struct {
	rdb *redis.Client
}

func NewImageTaskStore(rdb *redis.Client) service.ImageTaskStore {
	return &imageTaskStore{rdb: rdb}
}

func (s *imageTaskStore) Save(ctx context.Context, task *service.ImageTaskRecord, ttl time.Duration) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	index := fmt.Sprintf("image_task_index:%d:%d", task.UserID, task.APIKeyID)
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, imageTaskKey(task.ID), data, ttl)
	pipe.ZAdd(ctx, index, redis.Z{Score: float64(task.CreatedAt), Member: task.ID})
	pipe.ZRemRangeByScore(ctx, index, "-inf", fmt.Sprint(time.Now().Add(-service.ImageRetention-30*time.Minute).Unix()))
	pipe.Expire(ctx, index, ttl+30*time.Minute)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *imageTaskStore) List(ctx context.Context, owner service.ImageTaskOwner, limit int) ([]*service.ImageTaskRecord, error) {
	index := fmt.Sprintf("image_task_index:%d:%d", owner.UserID, owner.APIKeyID)
	ids, err := s.rdb.ZRevRange(ctx, index, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	result := make([]*service.ImageTaskRecord, 0, len(ids))
	for _, id := range ids {
		task, err := s.Get(ctx, id)
		if err == service.ErrImageTaskNotFound {
			_ = s.rdb.ZRem(ctx, index, id).Err()
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, nil
}

func (s *imageTaskStore) Get(ctx context.Context, id string) (*service.ImageTaskRecord, error) {
	data, err := s.rdb.Get(ctx, imageTaskKey(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, service.ErrImageTaskNotFound
		}
		return nil, err
	}
	var task service.ImageTaskRecord
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func imageTaskKey(id string) string {
	return imageTaskKeyPrefix + strings.TrimSpace(id)
}

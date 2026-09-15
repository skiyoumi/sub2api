package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type imageAssetRepository struct{ db *sql.DB }

func NewImageAssetRepository(db *sql.DB) service.ImageAssetRepository {
	return &imageAssetRepository{db: db}
}

func (r *imageAssetRepository) Create(ctx context.Context, a *service.ImageAssetRecord) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO image_assets (id,task_id,object_key,content_type,storage_config,expires_at,next_attempt_at) VALUES ($1,$2,$3,$4,$5,$6,$6)`, a.ID, a.TaskID, a.Key, a.ContentType, string(a.Config), a.ExpiresAt)
	return err
}

func (r *imageAssetRepository) Ready(ctx context.Context, id, version string, expires time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE image_assets SET ready=TRUE,version_id=$2,expires_at=$3,next_attempt_at=$3 WHERE id=$1`, id, version, expires)
	return err
}

const imageAssetColumns = `id,task_id,object_key,content_type,storage_config,version_id,ready,expires_at`

func scanImageAsset(row interface{ Scan(...any) error }) (*service.ImageAssetRecord, error) {
	a := &service.ImageAssetRecord{}
	err := row.Scan(&a.ID, &a.TaskID, &a.Key, &a.ContentType, &a.Config, &a.VersionID, &a.Ready, &a.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrImageTaskNotFound
	}
	return a, err
}

func (r *imageAssetRepository) Get(ctx context.Context, id string) (*service.ImageAssetRecord, error) {
	return scanImageAsset(r.db.QueryRowContext(ctx, `SELECT `+imageAssetColumns+` FROM image_assets WHERE id=$1`, id))
}

func (r *imageAssetRepository) Due(ctx context.Context, now time.Time, limit int) ([]*service.ImageAssetRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+imageAssetColumns+` FROM image_assets WHERE next_attempt_at <= $1 AND expires_at <= $1 ORDER BY next_attempt_at LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*service.ImageAssetRecord, 0)
	for rows.Next() {
		a, err := scanImageAsset(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *imageAssetRepository) Retry(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE image_assets SET next_attempt_at=$2 WHERE id=$1`, id, at)
	return err
}

func (r *imageAssetRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM image_assets WHERE id=$1`, id)
	return err
}

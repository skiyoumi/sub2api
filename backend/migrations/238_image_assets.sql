-- Durable deletion journal. Never expire these rows before deleting the object.
CREATE TABLE IF NOT EXISTS image_assets (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    object_key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_config JSONB NOT NULL,
    version_id TEXT NOT NULL DEFAULT '',
    ready BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    next_attempt_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS image_assets_cleanup_idx ON image_assets (next_attempt_at);

ALTER TABLE users ADD COLUMN IF NOT EXISTS registration_device_id VARCHAR(128) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_registration_device_id
    ON users (registration_device_id)
    WHERE deleted_at IS NULL AND registration_device_id <> '';

CREATE INDEX IF NOT EXISTS idx_users_registration_ip_created_at
    ON users (registration_ip, created_at)
    WHERE deleted_at IS NULL AND registration_ip <> '';

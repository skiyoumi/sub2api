CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_registration_ip_created_at
    ON users (registration_ip, created_at)
    WHERE deleted_at IS NULL AND registration_ip <> '';

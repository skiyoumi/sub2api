-- Recharge package order snapshots and expiring bonus wallet ledger.
ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS recharge_package_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS base_amount DECIMAL(20,2),
    ADD COLUMN IF NOT EXISTS permanent_credit_amount DECIMAL(20,8),
    ADD COLUMN IF NOT EXISTS bonus_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bonus_validity_days INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bonus_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS recharge_package_snapshot JSONB;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_orders_bonus_amount_nonnegative') THEN
        ALTER TABLE payment_orders ADD CONSTRAINT payment_orders_bonus_amount_nonnegative CHECK (bonus_amount >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_orders_bonus_validity_nonnegative') THEN
        ALTER TABLE payment_orders ADD CONSTRAINT payment_orders_bonus_validity_nonnegative CHECK (bonus_validity_days >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_orders_bonus_shape') THEN
        ALTER TABLE payment_orders ADD CONSTRAINT payment_orders_bonus_shape CHECK (
            (bonus_amount = 0 AND bonus_validity_days = 0 AND bonus_expires_at IS NULL)
            OR (bonus_amount > 0 AND bonus_validity_days > 0)
        );
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS wallet_bonus_grants (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    source_type VARCHAR(32) NOT NULL,
    source_id VARCHAR(64) NOT NULL,
    initial_amount DECIMAL(20,8) NOT NULL CHECK (initial_amount > 0),
    remaining_amount DECIMAL(20,8) NOT NULL CHECK (remaining_amount >= 0),
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CONSUMED', 'EXPIRED', 'REVOKED')),
    expired_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_type, source_id)
);

CREATE INDEX IF NOT EXISTS idx_wallet_bonus_grants_spend
    ON wallet_bonus_grants(user_id, expires_at, id)
    WHERE status = 'ACTIVE' AND remaining_amount > 0;
CREATE INDEX IF NOT EXISTS idx_wallet_bonus_grants_expire
    ON wallet_bonus_grants(expires_at, id)
    WHERE status = 'ACTIVE' AND remaining_amount > 0;

CREATE TABLE IF NOT EXISTS wallet_bonus_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    grant_id BIGINT NOT NULL REFERENCES wallet_bonus_grants(id),
    type VARCHAR(24) NOT NULL CHECK (type IN ('GRANT', 'SPEND', 'HOLD', 'CAPTURE', 'RELEASE', 'EXPIRE', 'REFUND_REVOKE', 'ADMIN_ADJUST')),
    amount DECIMAL(20,8) NOT NULL CHECK (amount > 0),
    request_id VARCHAR(128) NOT NULL,
    source_type VARCHAR(32) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    balance_after DECIMAL(20,8) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (request_id, grant_id, type)
);

CREATE INDEX IF NOT EXISTS idx_wallet_bonus_transactions_user_created
    ON wallet_bonus_transactions(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS wallet_hold_allocations (
    id BIGSERIAL PRIMARY KEY,
    hold_key VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    grant_id BIGINT REFERENCES wallet_bonus_grants(id),
    source_key VARCHAR(64) NOT NULL,
    amount DECIMAL(20,8) NOT NULL CHECK (amount > 0),
    captured_amount DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (captured_amount >= 0),
    released_amount DECIMAL(20,8) NOT NULL DEFAULT 0 CHECK (released_amount >= 0),
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CAPTURED', 'RELEASED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (hold_key, source_key),
    CHECK (captured_amount + released_amount <= amount),
    CHECK ((grant_id IS NULL AND source_key = 'permanent') OR (grant_id IS NOT NULL AND source_key <> 'permanent'))
);

CREATE INDEX IF NOT EXISTS idx_wallet_hold_allocations_user_status
    ON wallet_hold_allocations(user_id, status);

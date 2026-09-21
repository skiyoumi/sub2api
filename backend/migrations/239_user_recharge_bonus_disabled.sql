-- Preserve current eligibility by default. Existing wallet grants and orders are unchanged.
ALTER TABLE users ADD COLUMN IF NOT EXISTS recharge_bonus_disabled BOOLEAN NOT NULL DEFAULT FALSE;

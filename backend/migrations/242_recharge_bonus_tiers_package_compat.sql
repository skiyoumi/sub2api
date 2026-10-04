ALTER TABLE payment_orders DROP CONSTRAINT IF EXISTS payment_orders_bonus_shape;

ALTER TABLE payment_orders ADD CONSTRAINT payment_orders_bonus_shape CHECK (
    (bonus_amount = 0 AND bonus_validity_days = 0 AND bonus_expires_at IS NULL)
    OR (bonus_amount > 0 AND bonus_validity_days > 0)
    OR (bonus_amount > 0 AND recharge_package_id IS NULL AND bonus_validity_days = 0 AND bonus_expires_at IS NULL)
);

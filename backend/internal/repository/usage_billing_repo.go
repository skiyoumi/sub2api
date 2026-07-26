package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	return r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
}

func (r *usageBillingRepository) claimUsageBillingRequest(ctx context.Context, tx *sql.Tx, requestID string, apiKeyID int64, requestFingerprint string) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, requestID, apiKeyID, requestFingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = $1 AND api_key_id = $2
		`, requestID, apiKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) ReserveBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, reserveUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) CaptureBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, captureUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) ReleaseBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, releaseUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) applyBatchImageBalanceHold(
	ctx context.Context,
	cmd *service.BatchImageBalanceHoldCommand,
	apply func(context.Context, *sql.Tx, *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error),
) (_ *service.BatchImageBalanceHoldResult, err error) {
	if cmd == nil {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}
	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.BatchImageBalanceHoldResult{Applied: false}, nil
	}

	result, err := apply(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &service.BatchImageBalanceHoldResult{}
	}
	result.Applied = true

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil {
		if err := incrementUsageBillingSubscription(ctx, tx, *cmd.SubscriptionID, cmd.SubscriptionCost); err != nil {
			return err
		}
	}

	if cmd.BalanceCost > 0 {
		deduction, err := deductUsageBillingBalanceWithBonus(ctx, tx, cmd.UserID, cmd.BalanceCost, cmd.RequestID)
		if err != nil {
			return err
		}
		result.NewBalance = &deduction.NewBalance
		result.BalanceOverdrafted = !deduction.Sufficient
		result.BonusSpent = deduction.BonusSpent
		result.PermanentSpent = deduction.PermanentSpent
	}

	if cmd.APIKeyQuotaCost > 0 {
		exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
		if err != nil {
			return err
		}
		result.APIKeyQuotaExhausted = exhausted
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
	}

	if cmd.AccountQuotaCost > 0 && (strings.EqualFold(cmd.AccountType, service.AccountTypeAPIKey) || strings.EqualFold(cmd.AccountType, service.AccountTypeBedrock)) {
		quotaState, err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost)
		if err != nil {
			return err
		}
		result.QuotaState = quotaState
	}

	return nil
}

type usageBillingBalanceDeduction struct {
	NewBalance     float64
	Sufficient     bool
	BonusSpent     float64
	PermanentSpent float64
}

type lockedBonusGrant struct {
	id         int64
	remaining  decimal.Decimal
	expiresAt  string
	sourceType string
	sourceID   string
}

func deductUsageBillingBalanceWithBonus(ctx context.Context, tx *sql.Tx, userID int64, amount float64, requestID string) (*usageBillingBalanceDeduction, error) {
	var rawBalance string
	if err := tx.QueryRowContext(ctx, `
		SELECT balance::text FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, userID).Scan(&rawBalance); errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	} else if err != nil {
		return nil, err
	}
	currentBalance, err := decimal.NewFromString(rawBalance)
	if err != nil {
		return nil, fmt.Errorf("parse user balance: %w", err)
	}

	expiredRows, err := tx.QueryContext(ctx, `
		SELECT id, remaining_amount::text, source_type, source_id
		FROM wallet_bonus_grants
		WHERE user_id = $1 AND status = 'ACTIVE' AND remaining_amount > 0 AND expires_at <= NOW()
		ORDER BY expires_at, id
		FOR UPDATE`, userID)
	if err != nil {
		return nil, err
	}
	expired := decimal.Zero
	var expiredGrants []lockedBonusGrant
	for expiredRows.Next() {
		var grant lockedBonusGrant
		var raw string
		if err := expiredRows.Scan(&grant.id, &raw, &grant.sourceType, &grant.sourceID); err != nil {
			_ = expiredRows.Close()
			return nil, err
		}
		grant.remaining, err = decimal.NewFromString(raw)
		if err != nil {
			_ = expiredRows.Close()
			return nil, err
		}
		expired = expired.Add(grant.remaining)
		expiredGrants = append(expiredGrants, grant)
	}
	if err := expiredRows.Err(); err != nil {
		_ = expiredRows.Close()
		return nil, err
	}
	if err := expiredRows.Close(); err != nil {
		return nil, err
	}

	grantRows, err := tx.QueryContext(ctx, `
		SELECT id, remaining_amount::text, expires_at::text, source_type, source_id
		FROM wallet_bonus_grants
		WHERE user_id = $1 AND status = 'ACTIVE' AND remaining_amount > 0 AND expires_at > NOW()
		ORDER BY expires_at, id
		FOR UPDATE`, userID)
	if err != nil {
		return nil, err
	}
	var grants []lockedBonusGrant
	for grantRows.Next() {
		var grant lockedBonusGrant
		var raw string
		if err := grantRows.Scan(&grant.id, &raw, &grant.expiresAt, &grant.sourceType, &grant.sourceID); err != nil {
			_ = grantRows.Close()
			return nil, err
		}
		grant.remaining, err = decimal.NewFromString(raw)
		if err != nil {
			_ = grantRows.Close()
			return nil, err
		}
		grants = append(grants, grant)
	}
	if err := grantRows.Err(); err != nil {
		_ = grantRows.Close()
		return nil, err
	}
	if err := grantRows.Close(); err != nil {
		return nil, err
	}

	spendGrants := make([]service.BonusSpendGrant, 0, len(grants))
	for _, grant := range grants {
		spendGrants = append(spendGrants, service.BonusSpendGrant{GrantID: grant.id, Available: grant.remaining})
	}
	plan := service.BuildBonusSpendPlan(decimal.NewFromFloat(amount), spendGrants)
	effectiveBalance := currentBalance.Sub(expired)
	newBalance := effectiveBalance.Sub(decimal.NewFromFloat(amount))

	for _, grant := range expiredGrants {
		if _, err := tx.ExecContext(ctx, `
			UPDATE wallet_bonus_grants
			SET remaining_amount = 0, status = 'EXPIRED', expired_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND status = 'ACTIVE'`, grant.id); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO wallet_bonus_transactions
				(user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after)
			VALUES ($1, $2, 'EXPIRE', $3, $4, $5, $6, $7)
			ON CONFLICT (request_id, grant_id, type) DO NOTHING`, userID, grant.id,
			grant.remaining.String(), "expiry:inline:"+strconv.FormatInt(grant.id, 10), grant.sourceType, grant.sourceID, newBalance.String()); err != nil {
			return nil, err
		}
	}
	grantByID := make(map[int64]lockedBonusGrant, len(grants))
	for _, grant := range grants {
		grantByID[grant.id] = grant
	}
	for _, allocation := range plan.Allocations {
		grant := grantByID[allocation.GrantID]
		if _, err := tx.ExecContext(ctx, `
			UPDATE wallet_bonus_grants SET
				remaining_amount = remaining_amount - $1,
				status = CASE WHEN remaining_amount - $1 <= 0 THEN 'CONSUMED' ELSE status END,
				updated_at = NOW()
			WHERE id = $2 AND status = 'ACTIVE' AND remaining_amount >= $1`, allocation.Amount.String(), allocation.GrantID); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO wallet_bonus_transactions
				(user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after)
			VALUES ($1, $2, 'SPEND', $3, $4, $5, $6, $7)`, userID, allocation.GrantID,
			allocation.Amount.String(), requestID, grant.sourceType, grant.sourceID, newBalance.String()); err != nil {
			return nil, err
		}
	}
	var persistedBalance float64
	if err := tx.QueryRowContext(ctx, `
		UPDATE users SET balance = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance`, newBalance.String(), userID).Scan(&persistedBalance); err != nil {
		return nil, err
	}
	return &usageBillingBalanceDeduction{
		NewBalance: persistedBalance, Sufficient: effectiveBalance.GreaterThanOrEqual(decimal.NewFromFloat(amount)),
		BonusSpent: plan.Bonus.InexactFloat64(), PermanentSpent: plan.Permanent.InexactFloat64(),
	}, nil
}

func incrementUsageBillingSubscription(ctx context.Context, tx *sql.Tx, subscriptionID int64, costUSD float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			daily_usage_usd = us.daily_usage_usd + $1,
			weekly_usage_usd = us.weekly_usage_usd + $1,
			monthly_usage_usd = us.monthly_usage_usd + $1,
			updated_at = NOW()
		FROM groups g
		WHERE us.id = $2
			AND us.deleted_at IS NULL
			AND us.group_id = g.id
			AND g.deleted_at IS NULL
	`
	res, err := tx.ExecContext(ctx, updateSQL, costUSD, subscriptionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return service.ErrSubscriptionNotFound
}

func deductUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount float64) (float64, bool, error) {
	var newBalance float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance
	`, amount, userID).Scan(&newBalance)
	if err == nil {
		return newBalance, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance
	`, amount, userID).Scan(&newBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, service.ErrUserNotFound
	}
	if err != nil {
		return 0, false, err
	}
	return newBalance, false, nil
}

func reserveUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if strings.TrimSpace(cmd.BatchID) != "" {
		return reserveBatchImageBalanceWithBonus(ctx, tx, cmd)
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			frozen_balance = COALESCE(frozen_balance, 0) + $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, service.ErrBatchImageInsufficientBalance
}

func captureUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 && cmd.ActualAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if cmd.ActualAmount-cmd.HoldAmount > 0.00000001 {
		return nil, service.ErrBatchImageSettlementCostExceedsHold
	}
	if strings.TrimSpace(cmd.BatchID) != "" {
		return settleBatchImageBalanceWithBonus(ctx, tx, cmd, true)
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance
				+ CASE WHEN $1 > $2 THEN $1 - $2 ELSE 0 END
				- CASE WHEN $2 > $1 THEN $2 - $1 ELSE 0 END,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.ActualAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

func releaseUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if strings.TrimSpace(cmd.BatchID) != "" {
		return settleBatchImageBalanceWithBonus(ctx, tx, cmd, false)
	}
	// 释放前校验该 job 确实预留过 hold（hold request id 已被 claim），
	// 防止从未成功冻结的 job 触发"幻影释放"，从其他用户的冻结资金池中凭空生成余额。
	held, heldErr := batchImageHoldClaimExists(ctx, tx, service.BatchImageHoldRequestID(cmd.BatchID), cmd.APIKeyID)
	if heldErr != nil {
		return nil, heldErr
	}
	if !held {
		logger.LegacyPrintf("repository.usage_billing", "[BatchImage] release skipped, hold was never reserved: batch=%s", cmd.BatchID)
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

type batchImageHoldAllocation struct {
	id       int64
	grantID  sql.NullInt64
	amount   decimal.Decimal
	captured decimal.Decimal
	released decimal.Decimal
	expires  sql.NullTime
}

func reserveBatchImageBalanceWithBonus(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	hold := decimal.NewFromFloat(cmd.HoldAmount)
	var rawBalance, rawFrozen string
	if err := tx.QueryRowContext(ctx, `
		SELECT balance::text, COALESCE(frozen_balance, 0)::text
		FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, cmd.UserID).Scan(&rawBalance, &rawFrozen); errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	} else if err != nil {
		return nil, err
	}
	balance, err := decimal.NewFromString(rawBalance)
	if err != nil {
		return nil, err
	}
	frozen, err := decimal.NewFromString(rawFrozen)
	if err != nil {
		return nil, err
	}

	grants, expired, err := lockBatchImageBonusGrants(ctx, tx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	effective := balance.Sub(expired)
	if effective.LessThan(hold) {
		return nil, service.ErrBatchImageInsufficientBalance
	}
	newBalance := effective.Sub(hold)
	newFrozen := frozen.Add(hold)
	if err := expireLockedBonusGrants(ctx, tx, cmd.UserID, grants.expired, newBalance); err != nil {
		return nil, err
	}

	spendGrants := make([]service.BonusSpendGrant, 0, len(grants.active))
	for _, grant := range grants.active {
		spendGrants = append(spendGrants, service.BonusSpendGrant{GrantID: grant.id, Available: grant.remaining})
	}
	plan := service.BuildBonusSpendPlan(hold, spendGrants)
	grantByID := make(map[int64]lockedBonusGrant, len(grants.active))
	for _, grant := range grants.active {
		grantByID[grant.id] = grant
	}
	for _, allocation := range plan.Allocations {
		grant := grantByID[allocation.GrantID]
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_bonus_grants SET remaining_amount = remaining_amount - $1, updated_at = NOW() WHERE id = $2 AND status = 'ACTIVE' AND remaining_amount >= $1`, allocation.Amount.String(), allocation.GrantID); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_hold_allocations (hold_key, user_id, grant_id, source_key, amount) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (hold_key, source_key) DO NOTHING`, cmd.BatchID, cmd.UserID, allocation.GrantID, strconv.FormatInt(allocation.GrantID, 10), allocation.Amount.String()); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_bonus_transactions (user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after) VALUES ($1, $2, 'HOLD', $3, $4, $5, $6, $7) ON CONFLICT (request_id, grant_id, type) DO NOTHING`, cmd.UserID, allocation.GrantID, allocation.Amount.String(), cmd.RequestID, grant.sourceType, grant.sourceID, newBalance.String()); err != nil {
			return nil, err
		}
	}
	if plan.Permanent.IsPositive() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_hold_allocations (hold_key, user_id, grant_id, source_key, amount) VALUES ($1, $2, NULL, 'permanent', $3) ON CONFLICT (hold_key, source_key) DO NOTHING`, cmd.BatchID, cmd.UserID, plan.Permanent.String()); err != nil {
			return nil, err
		}
	}
	var persistedBalance, persistedFrozen float64
	if err := tx.QueryRowContext(ctx, `UPDATE users SET balance = $1, frozen_balance = $2, updated_at = NOW() WHERE id = $3 RETURNING balance, frozen_balance`, newBalance.String(), newFrozen.String(), cmd.UserID).Scan(&persistedBalance, &persistedFrozen); err != nil {
		return nil, err
	}
	return &service.BatchImageBalanceHoldResult{NewBalance: &persistedBalance, FrozenBalance: &persistedFrozen}, nil
}

type batchImageLockedGrants struct{ active, expired []lockedBonusGrant }

func lockBatchImageBonusGrants(ctx context.Context, tx *sql.Tx, userID int64) (batchImageLockedGrants, decimal.Decimal, error) {
	var result batchImageLockedGrants
	expiredTotal := decimal.Zero
	rows, err := tx.QueryContext(ctx, `SELECT id, remaining_amount::text, source_type, source_id, expires_at <= NOW() FROM wallet_bonus_grants WHERE user_id = $1 AND status = 'ACTIVE' AND remaining_amount > 0 ORDER BY expires_at, id FOR UPDATE`, userID)
	if err != nil {
		return result, expiredTotal, err
	}
	defer rows.Close()
	for rows.Next() {
		var grant lockedBonusGrant
		var raw string
		var expired bool
		if err := rows.Scan(&grant.id, &raw, &grant.sourceType, &grant.sourceID, &expired); err != nil {
			return result, expiredTotal, err
		}
		grant.remaining, err = decimal.NewFromString(raw)
		if err != nil {
			return result, expiredTotal, err
		}
		if expired {
			result.expired = append(result.expired, grant)
			expiredTotal = expiredTotal.Add(grant.remaining)
		} else {
			result.active = append(result.active, grant)
		}
	}
	return result, expiredTotal, rows.Err()
}

func expireLockedBonusGrants(ctx context.Context, tx *sql.Tx, userID int64, grants []lockedBonusGrant, balanceAfter decimal.Decimal) error {
	for _, grant := range grants {
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_bonus_grants SET remaining_amount = 0, status = 'EXPIRED', expired_at = NOW(), updated_at = NOW() WHERE id = $1 AND status = 'ACTIVE'`, grant.id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_bonus_transactions (user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after) VALUES ($1, $2, 'EXPIRE', $3, $4, $5, $6, $7) ON CONFLICT (request_id, grant_id, type) DO NOTHING`, userID, grant.id, grant.remaining.String(), "expiry:inline:"+strconv.FormatInt(grant.id, 10), grant.sourceType, grant.sourceID, balanceAfter.String()); err != nil {
			return err
		}
	}
	return nil
}

func settleBatchImageBalanceWithBonus(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand, capture bool) (*service.BatchImageBalanceHoldResult, error) {
	var rawBalance, rawFrozen string
	if err := tx.QueryRowContext(ctx, `SELECT balance::text, COALESCE(frozen_balance, 0)::text FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, cmd.UserID).Scan(&rawBalance, &rawFrozen); errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	} else if err != nil {
		return nil, err
	}
	balance, err := decimal.NewFromString(rawBalance)
	if err != nil {
		return nil, err
	}
	frozen, err := decimal.NewFromString(rawFrozen)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT a.id, a.grant_id, a.amount::text, a.captured_amount::text, a.released_amount::text, g.expires_at FROM wallet_hold_allocations a LEFT JOIN wallet_bonus_grants g ON g.id = a.grant_id WHERE a.hold_key = $1 AND a.user_id = $2 AND a.status = 'ACTIVE' ORDER BY a.id FOR UPDATE`, cmd.BatchID, cmd.UserID)
	if err != nil {
		return nil, err
	}
	var allocations []batchImageHoldAllocation
	total := decimal.Zero
	for rows.Next() {
		var a batchImageHoldAllocation
		var amount, capturedAmount, releasedAmount string
		if err := rows.Scan(&a.id, &a.grantID, &amount, &capturedAmount, &releasedAmount, &a.expires); err != nil {
			_ = rows.Close()
			return nil, err
		}
		a.amount, err = decimal.NewFromString(amount)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		a.captured, err = decimal.NewFromString(capturedAmount)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		a.released, err = decimal.NewFromString(releasedAmount)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		total = total.Add(a.amount.Sub(a.captured).Sub(a.released))
		allocations = append(allocations, a)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	if len(allocations) == 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	actual := decimal.Zero
	if capture {
		actual = decimal.NewFromFloat(cmd.ActualAmount)
	}
	if actual.GreaterThan(total) {
		return nil, service.ErrBatchImageSettlementCostExceedsHold
	}
	restore := decimal.Zero
	remainingCapture := actual
	now := time.Now()
	for _, a := range allocations {
		available := a.amount.Sub(a.captured).Sub(a.released)
		captured := decimal.Min(available, remainingCapture)
		released := available.Sub(captured)
		remainingCapture = remainingCapture.Sub(captured)
		restored := released
		if a.grantID.Valid {
			if captured.IsPositive() {
				if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_bonus_transactions (user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after) SELECT $1, id, 'CAPTURE', $2, $3, source_type, source_id, $4 FROM wallet_bonus_grants WHERE id = $5 ON CONFLICT (request_id, grant_id, type) DO NOTHING`, cmd.UserID, captured.String(), cmd.RequestID, balance.String(), a.grantID.Int64); err != nil {
					return nil, err
				}
			}
			if released.IsPositive() && a.expires.Valid && a.expires.Time.After(now) {
				if _, err := tx.ExecContext(ctx, `UPDATE wallet_bonus_grants SET remaining_amount = remaining_amount + $1, status = 'ACTIVE', updated_at = NOW() WHERE id = $2`, released.String(), a.grantID.Int64); err != nil {
					return nil, err
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_bonus_transactions (user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after) SELECT $1, id, 'RELEASE', $2, $3, source_type, source_id, $4 FROM wallet_bonus_grants WHERE id = $5 ON CONFLICT (request_id, grant_id, type) DO NOTHING`, cmd.UserID, released.String(), cmd.RequestID, balance.String(), a.grantID.Int64); err != nil {
					return nil, err
				}
			} else if released.IsPositive() {
				restored = decimal.Zero
				if _, err := tx.ExecContext(ctx, `UPDATE wallet_bonus_grants SET status = CASE WHEN remaining_amount = 0 THEN 'EXPIRED' ELSE status END, expired_at = COALESCE(expired_at, NOW()), updated_at = NOW() WHERE id = $1`, a.grantID.Int64); err != nil {
					return nil, err
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_bonus_transactions (user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after) SELECT $1, id, 'EXPIRE', $2, $3, source_type, source_id, $4 FROM wallet_bonus_grants WHERE id = $5 ON CONFLICT (request_id, grant_id, type) DO NOTHING`, cmd.UserID, released.String(), cmd.RequestID, balance.String(), a.grantID.Int64); err != nil {
					return nil, err
				}
			}
		}
		restore = restore.Add(restored)
		status := "RELEASED"
		if captured.IsPositive() {
			status = "CAPTURED"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_hold_allocations SET captured_amount = captured_amount + $1, released_amount = released_amount + $2, status = $3, updated_at = NOW() WHERE id = $4`, captured.String(), released.String(), status, a.id); err != nil {
			return nil, err
		}
		if a.grantID.Valid && captured.IsPositive() {
			if _, err := tx.ExecContext(ctx, `UPDATE wallet_bonus_grants g SET status = 'CONSUMED', updated_at = NOW() WHERE g.id = $1 AND g.status = 'ACTIVE' AND g.remaining_amount = 0 AND NOT EXISTS (SELECT 1 FROM wallet_hold_allocations a WHERE a.grant_id = g.id AND a.status = 'ACTIVE')`, a.grantID.Int64); err != nil {
				return nil, err
			}
		}
	}
	newBalance := balance.Add(restore)
	newFrozen := frozen.Sub(total)
	var persistedBalance, persistedFrozen float64
	if err := tx.QueryRowContext(ctx, `UPDATE users SET balance = $1, frozen_balance = $2, updated_at = NOW() WHERE id = $3 AND COALESCE(frozen_balance, 0) >= $4 RETURNING balance, frozen_balance`, newBalance.String(), newFrozen.String(), cmd.UserID, total.String()).Scan(&persistedBalance, &persistedFrozen); err != nil {
		return nil, err
	}
	return &service.BatchImageBalanceHoldResult{NewBalance: &persistedBalance, FrozenBalance: &persistedFrozen}, nil
}

// batchImageHoldClaimExists 检查 hold request id 是否已在 dedup（或归档）表中被 claim，
// 即该 batch 的冻结操作确实成功提交过。
func batchImageHoldClaimExists(ctx context.Context, tx *sql.Tx, holdRequestID string, apiKeyID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup
		WHERE request_id = $1 AND api_key_id = $2
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	err = tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func userExistsForBilling(ctx context.Context, tx *sql.Tx, userID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var exhausted bool
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0
					AND status = $3
					AND quota_used < quota
					AND quota_used + $1 >= quota
				THEN $4
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING quota > 0 AND quota_used >= quota AND quota_used - $1 < quota
	`, amount, apiKeyID, service.StatusAPIKeyActive, service.StatusAPIKeyQuotaExhausted).Scan(&exhausted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return exhausted, nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) (*service.AccountQuotaState, error) {
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN `+dailyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN `+dailyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+dailyExpiredExpr+` AND `+nextDailyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_daily_reset_at', `+nextDailyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+weeklyExpiredExpr+` AND `+nextWeeklyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_weekly_reset_at', `+nextWeeklyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0),
			COALESCE((extra->>'quota_daily_used')::numeric, 0),
			COALESCE((extra->>'quota_daily_limit')::numeric, 0),
			COALESCE((extra->>'quota_weekly_used')::numeric, 0),
			COALESCE((extra->>'quota_weekly_limit')::numeric, 0)`,
		amount, accountID)
	if err != nil {
		return nil, err
	}

	var state service.AccountQuotaState
	if rows.Next() {
		if err := rows.Scan(
			&state.TotalUsed, &state.TotalLimit,
			&state.DailyUsed, &state.DailyLimit,
			&state.WeeklyUsed, &state.WeeklyLimit,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
	} else {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	// 必须在执行下一条 SQL 前显式关闭 rows：pq 驱动在同一连接上
	// 不允许前一条查询的结果集未耗尽时启动新查询，否则会返回
	// "unexpected Parse response" 错误。
	if err := rows.Close(); err != nil {
		return nil, err
	}
	// 任意维度额度在本次递增中从"未超"跨越到"已超"时，必须刷新调度快照，
	// 否则 Redis 中缓存的 Account 仍显示旧的 used 值，后续请求会继续选中本账号，
	// 最终观察到 daily_used / weekly_used 大幅超过配置的 limit。
	// 对于日/周额度，即使本次触发了周期重置（pre=0、post=amount），
	// 判定式 (post-amount) < limit 同样成立，逻辑与总额度保持一致。
	crossedTotal := state.TotalLimit > 0 && state.TotalUsed >= state.TotalLimit && (state.TotalUsed-amount) < state.TotalLimit
	crossedDaily := state.DailyLimit > 0 && state.DailyUsed >= state.DailyLimit && (state.DailyUsed-amount) < state.DailyLimit
	crossedWeekly := state.WeeklyLimit > 0 && state.WeeklyUsed >= state.WeeklyLimit && (state.WeeklyUsed-amount) < state.WeeklyLimit
	if crossedTotal || crossedDaily || crossedWeekly {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return nil, err
		}
	}
	return &state, nil
}

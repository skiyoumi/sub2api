package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

const bonusGrantSourcePaymentOrder = "payment_order"

type BonusGrantInput struct {
	UserID          int64
	PaymentOrderID  int64
	PermanentAmount decimal.Decimal
	BonusAmount     decimal.Decimal
	ValidityDays    int
	FulfilledAt     time.Time
}

type BonusGrantResult struct {
	GrantID        int64
	BonusExpiresAt time.Time
	Applied        bool
}

type BonusWalletSummary struct {
	Balance       decimal.Decimal
	NearestExpiry *time.Time
}

type BonusSpendGrant struct {
	GrantID   int64
	ExpiresAt time.Time
	Available decimal.Decimal
}

type BonusSpendAllocation struct {
	GrantID int64
	Amount  decimal.Decimal
}

type BonusSpendPlan struct {
	Bonus       decimal.Decimal
	Permanent   decimal.Decimal
	Allocations []BonusSpendAllocation
}

type BonusRevokeInput struct {
	UserID         int64
	PaymentOrderID int64
	RefundAmount   decimal.Decimal
	Force          bool
}

type BonusRevokeResult struct {
	TargetBonus decimal.Decimal
	Revoked     decimal.Decimal
	Uncovered   decimal.Decimal
}

type BonusExpireResult struct {
	Grants int
	Amount decimal.Decimal
	Users  int
}

// BonusWallet owns all mutations that introduce expiring payment bonuses.
// Consumption and hold integration use the same FEFO planner but are wired in
// subsequent repository changes so usage billing keeps its existing dedup tx.
type BonusWallet struct {
	client *dbent.Client
}

func NewBonusWallet(client *dbent.Client) *BonusWallet {
	return &BonusWallet{client: client}
}

func (w *BonusWallet) GrantForPayment(ctx context.Context, input BonusGrantInput) (*BonusGrantResult, error) {
	if w == nil || w.client == nil {
		return nil, errors.New("bonus wallet is not configured")
	}
	if input.UserID <= 0 || input.PaymentOrderID <= 0 || !input.PermanentAmount.IsPositive() || !input.BonusAmount.IsPositive() {
		return nil, fmt.Errorf("invalid payment bonus grant input")
	}
	if input.ValidityDays < 1 || input.ValidityDays > 3650 {
		return nil, fmt.Errorf("invalid bonus validity days")
	}
	fulfilledAt := input.FulfilledAt.UTC().Truncate(time.Microsecond)
	if fulfilledAt.IsZero() {
		fulfilledAt = time.Now().UTC().Truncate(time.Microsecond)
	}
	expiresAt := fulfilledAt.Add(time.Duration(input.ValidityDays) * 24 * time.Hour)

	tx, err := w.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin bonus grant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()

	var currentBalance string
	rows, err := client.QueryContext(ctx, `SELECT balance::text FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("lock bonus wallet user: %w", err)
	}
	if rows.Next() {
		err = rows.Scan(&currentBalance)
	} else {
		err = sql.ErrNoRows
	}
	_ = rows.Close()
	if err != nil {
		return nil, fmt.Errorf("lock bonus wallet user: %w", err)
	}

	requestID := "payment_order:" + strconv.FormatInt(input.PaymentOrderID, 10) + ":fulfill"
	var grantID int64
	grantRows, err := client.QueryContext(ctx, `
		INSERT INTO wallet_bonus_grants (
			user_id, source_type, source_id, initial_amount, remaining_amount, expires_at
		) VALUES ($1, $2, $3, $4, $4, $5)
		ON CONFLICT (source_type, source_id) DO NOTHING
		RETURNING id`, input.UserID, bonusGrantSourcePaymentOrder, strconv.FormatInt(input.PaymentOrderID, 10), input.BonusAmount.String(), expiresAt)
	if err != nil {
		return nil, fmt.Errorf("create payment bonus grant: %w", err)
	}
	inserted := grantRows.Next()
	if inserted {
		err = grantRows.Scan(&grantID)
	}
	_ = grantRows.Close()
	if err != nil {
		return nil, fmt.Errorf("read payment bonus grant: %w", err)
	}
	if !inserted {
		existingRows, queryErr := client.QueryContext(ctx, `
			SELECT id, expires_at FROM wallet_bonus_grants
			WHERE source_type = $1 AND source_id = $2`, bonusGrantSourcePaymentOrder, strconv.FormatInt(input.PaymentOrderID, 10))
		if queryErr != nil {
			return nil, fmt.Errorf("load existing payment bonus grant: %w", queryErr)
		}
		if existingRows.Next() {
			queryErr = existingRows.Scan(&grantID, &expiresAt)
		} else {
			queryErr = sql.ErrNoRows
		}
		_ = existingRows.Close()
		if queryErr != nil {
			return nil, fmt.Errorf("load existing payment bonus grant: %w", queryErr)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit existing payment bonus grant lookup: %w", err)
		}
		return &BonusGrantResult{GrantID: grantID, BonusExpiresAt: expiresAt, Applied: false}, nil
	}

	credit := input.PermanentAmount.Add(input.BonusAmount)
	var balanceAfter string
	updatedRows, err := client.QueryContext(ctx, `
		UPDATE users SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance::text`, credit.String(), input.UserID)
	if err != nil {
		return nil, fmt.Errorf("credit payment bonus wallet: %w", err)
	}
	if updatedRows.Next() {
		err = updatedRows.Scan(&balanceAfter)
	} else {
		err = sql.ErrNoRows
	}
	_ = updatedRows.Close()
	if err != nil {
		return nil, fmt.Errorf("credit payment bonus wallet: %w", err)
	}
	if _, err := client.ExecContext(ctx, `
		INSERT INTO wallet_bonus_transactions (
			user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after
		) VALUES ($1, $2, 'GRANT', $3, $4, $5, $6, $7)`,
		input.UserID, grantID, input.BonusAmount.String(), requestID,
		bonusGrantSourcePaymentOrder, strconv.FormatInt(input.PaymentOrderID, 10), balanceAfter); err != nil {
		return nil, fmt.Errorf("record payment bonus grant: %w", err)
	}
	if _, err := client.ExecContext(ctx, `
		UPDATE payment_orders SET bonus_expires_at = $1, updated_at = NOW() WHERE id = $2`,
		expiresAt, input.PaymentOrderID); err != nil {
		return nil, fmt.Errorf("persist payment bonus expiry: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit payment bonus grant: %w", err)
	}
	return &BonusGrantResult{GrantID: grantID, BonusExpiresAt: expiresAt, Applied: true}, nil
}

func (w *BonusWallet) GetSummary(ctx context.Context, userID int64) (*BonusWalletSummary, error) {
	if w == nil || w.client == nil {
		return nil, errors.New("bonus wallet is not configured")
	}
	rows, err := w.client.QueryContext(ctx, `
		SELECT COALESCE(SUM(remaining_amount), 0)::text, MIN(expires_at)
		FROM wallet_bonus_grants
		WHERE user_id = $1 AND status = 'ACTIVE' AND remaining_amount > 0 AND expires_at > NOW()`, userID)
	if err != nil {
		return nil, fmt.Errorf("query bonus wallet summary: %w", err)
	}
	defer rows.Close()
	var raw string
	var nearest sql.NullTime
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	if err := rows.Scan(&raw, &nearest); err != nil {
		return nil, fmt.Errorf("scan bonus wallet summary: %w", err)
	}
	balance, err := decimal.NewFromString(raw)
	if err != nil {
		return nil, fmt.Errorf("parse bonus wallet summary: %w", err)
	}
	result := &BonusWalletSummary{Balance: balance}
	if nearest.Valid {
		value := nearest.Time
		result.NearestExpiry = &value
	}
	return result, nil
}

func (s *PaymentService) GetBonusWalletSummary(ctx context.Context, userID int64) (*BonusWalletSummary, error) {
	if s == nil || s.bonusWallet == nil {
		return &BonusWalletSummary{Balance: decimal.Zero}, nil
	}
	return s.bonusWallet.GetSummary(ctx, userID)
}

func (s *PaymentService) ExpireBonusWalletGrants(ctx context.Context, limit int) (*BonusExpireResult, error) {
	if s == nil || s.bonusWallet == nil {
		return &BonusExpireResult{}, nil
	}
	return s.bonusWallet.ExpireBatch(ctx, limit)
}

// RevokeForRefund removes the proportional, still-available bonus for an order.
// The caller deducts the full refund amount from users.balance; this method only
// adjusts the grant ledger so the aggregate balance invariant remains true.
func (w *BonusWallet) RevokeForRefund(ctx context.Context, input BonusRevokeInput) (*BonusRevokeResult, error) {
	if w == nil || w.client == nil {
		return nil, errors.New("bonus wallet is not configured")
	}
	if input.UserID <= 0 || input.PaymentOrderID <= 0 || !input.RefundAmount.IsPositive() {
		return nil, fmt.Errorf("invalid bonus refund input")
	}
	tx, err := w.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin bonus refund transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	var bonus, credited decimal.Decimal
	var rawBonus, rawAmount string
	orderRows, err := client.QueryContext(ctx, `
		SELECT COALESCE(bonus_amount, 0)::text, amount::text
		FROM payment_orders WHERE id = $1 AND user_id = $2 FOR UPDATE`, input.PaymentOrderID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("load refund order snapshot: %w", err)
	}
	if !orderRows.Next() {
		_ = orderRows.Close()
		return nil, fmt.Errorf("load refund order snapshot: %w", sql.ErrNoRows)
	}
	if err := orderRows.Scan(&rawBonus, &rawAmount); err != nil {
		_ = orderRows.Close()
		return nil, fmt.Errorf("load refund order snapshot: %w", err)
	}
	if err := orderRows.Close(); err != nil {
		return nil, err
	}
	bonus, err = decimal.NewFromString(rawBonus)
	if err != nil {
		return nil, fmt.Errorf("parse refund bonus: %w", err)
	}
	credited, err = decimal.NewFromString(rawAmount)
	if err != nil {
		return nil, fmt.Errorf("parse refund credited amount: %w", err)
	}
	result := &BonusRevokeResult{}
	if !bonus.IsPositive() || !credited.IsPositive() {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	result.TargetBonus = CalculateBonusRefundTarget(credited, bonus, input.RefundAmount)
	if !result.TargetBonus.IsPositive() {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	var grantID int64
	var rawRemaining, sourceType, sourceID string
	grantRows, err := client.QueryContext(ctx, `
		SELECT id, remaining_amount::text, source_type, source_id
		FROM wallet_bonus_grants
		WHERE user_id = $1 AND source_type = $2 AND source_id = $3
		FOR UPDATE`, input.UserID, bonusGrantSourcePaymentOrder, strconv.FormatInt(input.PaymentOrderID, 10))
	if err != nil {
		return nil, fmt.Errorf("lock refund bonus grant: %w", err)
	}
	if !grantRows.Next() {
		_ = grantRows.Close()
		result.Uncovered = result.TargetBonus
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	err = grantRows.Scan(&grantID, &rawRemaining, &sourceType, &sourceID)
	if closeErr := grantRows.Close(); err == nil {
		err = closeErr
	}
	if errors.Is(err, sql.ErrNoRows) {
		result.Uncovered = result.TargetBonus
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lock refund bonus grant: %w", err)
	}
	remaining, err := decimal.NewFromString(rawRemaining)
	if err != nil {
		return nil, err
	}
	revoke := decimal.Min(remaining, result.TargetBonus)
	if revoke.IsPositive() {
		updateResult, err := client.ExecContext(ctx, `
			UPDATE wallet_bonus_grants SET remaining_amount = remaining_amount - $1,
			status = CASE WHEN remaining_amount - $1 <= 0 THEN 'REVOKED' ELSE status END,
			revoked_at = CASE WHEN remaining_amount - $1 <= 0 THEN NOW() ELSE revoked_at END,
			updated_at = NOW() WHERE id = $2 AND status = 'ACTIVE'`, revoke.String(), grantID)
		if err != nil {
			return nil, err
		}
		affected, err := updateResult.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected == 0 {
			revoke = decimal.Zero
		}
		requestID := fmt.Sprintf("refund:%d", input.PaymentOrderID)
		if revoke.IsPositive() {
			if _, err := client.ExecContext(ctx, `
			INSERT INTO wallet_bonus_transactions
			(user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after)
			VALUES ($1, $2, 'REFUND_REVOKE', $3, $4, $5, $6,
				(SELECT balance::text FROM users WHERE id = $1))
			ON CONFLICT (request_id, grant_id, type) DO NOTHING`, input.UserID, grantID, revoke.String(), requestID, sourceType, sourceID); err != nil {
				return nil, err
			}
		}
		result.Revoked = revoke
	}
	result.Uncovered = result.TargetBonus.Sub(result.Revoked)
	if result.Uncovered.IsNegative() {
		result.Uncovered = decimal.Zero
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit bonus refund: %w", err)
	}
	return result, nil
}

// ExpireBatch atomically expires at most limit grants. It is safe to run from
// multiple workers because rows are claimed with SKIP LOCKED and state changes
// are conditional.
func (w *BonusWallet) ExpireBatch(ctx context.Context, limit int) (*BonusExpireResult, error) {
	if w == nil || w.client == nil {
		return nil, errors.New("bonus wallet is not configured")
	}
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	tx, err := w.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id, remaining_amount::text, source_type, source_id
		FROM wallet_bonus_grants
		WHERE status = 'ACTIVE' AND remaining_amount > 0 AND expires_at <= NOW()
		ORDER BY expires_at, id LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	type grant struct {
		id, userID           int64
		amount               decimal.Decimal
		sourceType, sourceID string
	}
	var grants []grant
	for rows.Next() {
		var g grant
		var raw string
		if err := rows.Scan(&g.id, &g.userID, &raw, &g.sourceType, &g.sourceID); err != nil {
			_ = rows.Close()
			return nil, err
		}
		g.amount, err = decimal.NewFromString(raw)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		grants = append(grants, g)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	result := &BonusExpireResult{}
	users := map[int64]struct{}{}
	for _, g := range grants {
		var balance string
		userRows, err := tx.QueryContext(ctx, `SELECT balance::text FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, g.userID)
		if err != nil {
			return nil, err
		}
		if !userRows.Next() {
			_ = userRows.Close()
			return nil, sql.ErrNoRows
		}
		if err := userRows.Scan(&balance); err != nil {
			_ = userRows.Close()
			return nil, err
		}
		if err := userRows.Close(); err != nil {
			return nil, err
		}
		current, parseErr := decimal.NewFromString(balance)
		if parseErr != nil {
			return nil, parseErr
		}
		deduct := decimal.Min(current, g.amount)
		if deduct.IsPositive() {
			if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, deduct.String(), g.userID); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_bonus_grants SET remaining_amount = 0, status = 'EXPIRED', expired_at = NOW(), updated_at = NOW() WHERE id = $1 AND status = 'ACTIVE'`, g.id); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_bonus_transactions (user_id, grant_id, type, amount, request_id, source_type, source_id, balance_after) VALUES ($1, $2, 'EXPIRE', $3, $4, $5, $6, (SELECT balance::text FROM users WHERE id = $1)) ON CONFLICT (request_id, grant_id, type) DO NOTHING`, g.userID, g.id, g.amount.String(), fmt.Sprintf("expiry:%d", g.id), g.sourceType, g.sourceID); err != nil {
			return nil, err
		}
		result.Grants++
		result.Amount = result.Amount.Add(deduct)
		users[g.userID] = struct{}{}
	}
	result.Users = len(users)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func BuildBonusSpendPlan(amount decimal.Decimal, grants []BonusSpendGrant) BonusSpendPlan {
	remaining := amount
	plan := BonusSpendPlan{Bonus: decimal.Zero, Permanent: decimal.Zero}
	for _, grant := range grants {
		if !remaining.IsPositive() {
			break
		}
		available := grant.Available
		if !available.IsPositive() {
			continue
		}
		allocated := decimal.Min(available, remaining)
		plan.Allocations = append(plan.Allocations, BonusSpendAllocation{GrantID: grant.GrantID, Amount: allocated})
		plan.Bonus = plan.Bonus.Add(allocated)
		remaining = remaining.Sub(allocated)
	}
	plan.Permanent = remaining
	return plan
}

func CalculateBonusRefundTarget(credited, bonus, refund decimal.Decimal) decimal.Decimal {
	if !credited.IsPositive() || !bonus.IsPositive() || !refund.IsPositive() {
		return decimal.Zero
	}
	ratio := refund.Div(credited)
	if ratio.GreaterThan(decimal.NewFromInt(1)) {
		ratio = decimal.NewFromInt(1)
	}
	return bonus.Mul(ratio)
}

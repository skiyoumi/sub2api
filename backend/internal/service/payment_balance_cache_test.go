//go:build unit

package service

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rechargeAuthCache struct {
	authCacheStub
	entries sync.Map
}

func (c *rechargeAuthCache) GetAuthCache(ctx context.Context, key string) (*APIKeyAuthCacheEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if entry, ok := c.entries.Load(key); ok {
		return entry.(*APIKeyAuthCacheEntry), nil
	}
	return nil, redis.Nil
}

func (c *rechargeAuthCache) SetAuthCache(ctx context.Context, key string, entry *APIKeyAuthCacheEntry, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.entries.Store(key, entry)
	return nil
}

func (c *rechargeAuthCache) DeleteAuthCache(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.entries.Delete(key)
	return nil
}

// Exercise payment fulfillment and both real cache-reading services. The
// repository and Redis substitutes retain the pre-deposit zero balance until
// fulfillment invalidates it; no TTL advancement or eventual assertion is used.
func TestBalanceRechargeRefreshesCachedEligibility(t *testing.T) {
	for _, tt := range []struct {
		name      string
		bonus     bool
		recovered bool
	}{
		{name: "ordinary"},
		{name: "already_redeemed", recovered: true},
		{name: "bonus_package", bonus: true},
		{name: "already_granted_bonus", bonus: true, recovered: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := newPaymentConfigServiceTestClient(t)
			ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
			order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, OrderStatusPaid, time.Now())
			update := client.PaymentOrder.UpdateOneID(order.ID).
				SetOrderType(payment.OrderTypeBalance).
				ClearPlanID().ClearSubscriptionGroupID().ClearSubscriptionDays()
			credited := 80.0
			if tt.bonus {
				credited = 90
				update.SetAmount(credited).SetProviderSnapshot(map[string]any{
					"_recharge_package": map[string]any{
						"id": "bonus", "amount": "80", "permanent_credit_amount": "80",
						"bonus_amount": "10", "bonus_validity_days": 30,
					},
				})
			}
			order, err := update.Save(ctx)
			require.NoError(t, err)

			user := &User{ID: order.UserID, Status: StatusActive, Role: RoleUser}
			userRepo := &mockUserRepo{getByIDUser: user}
			keys := []string{"recharge-key-one", "recharge-key-two"}
			authRepo := &authRepoStub{
				getByKeyForAuth: func(_ context.Context, key string) (*APIKey, error) {
					copyUser := *user
					return &APIKey{ID: 1, UserID: user.ID, Key: key, Status: StatusActive, User: &copyUser}, nil
				},
				listKeysByUserID: func(ctx context.Context, id int64) ([]string, error) {
					require.Equal(t, user.ID, id)
					return keys, ctx.Err()
				},
			}
			cfg := &config.Config{APIKeyAuth: config.APIKeyAuthCacheConfig{L2TTLSeconds: 300}}
			authService := NewAPIKeyService(authRepo, nil, nil, nil, nil, &rechargeAuthCache{}, cfg)
			for _, key := range keys {
				cached, err := authService.GetByKey(ctx, key)
				require.NoError(t, err)
				require.Zero(t, cached.User.Balance)
			}
			balanceCache := &balanceEligibilityCacheStub{cacheMissAfterInvalidate: true}
			billing := NewBillingCacheService(balanceCache, userRepo, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billing.Stop)
			require.ErrorIs(t, billing.CheckBillingEligibility(ctx, user, nil, nil, nil, ""), ErrInsufficientBalance)

			redeemRepo := &paymentFulfillmentRedeemRepo{}
			if tt.recovered && !tt.bonus {
				usedBy := user.ID
				redeemRepo.codesByCode = map[string]*RedeemCode{order.RechargeCode: {
					ID: 101, Code: order.RechargeCode, Type: RedeemTypeBalance,
					Value: order.Amount, Status: StatusUsed, UsedBy: &usedBy,
				}}
			}
			userRepo.updateBalanceFn = func(_ context.Context, id int64, amount float64) error {
				require.Equal(t, user.ID, id)
				require.Equal(t, credited, amount)
				return nil
			}
			redeem := NewRedeemService(redeemRepo, userRepo, nil, nil, billing, client, authService, nil)
			svc := &PaymentService{entClient: client, redeemService: redeem, userRepo: userRepo}
			if tt.bonus {
				svc.bonusWallet = paymentCacheTestBonusWallet(t, order, tt.recovered)
			}

			require.NoError(t, svc.ExecuteBalanceFulfillment(ctx, order.ID))
			// The wallet SQL mock below confirms the committed balance. Reflect
			// that value in the user repository used by subsequent requests.
			user.Balance = credited
			for _, key := range keys {
				cached, err := authService.GetByKey(ctx, key)
				require.NoError(t, err)
				assert.Equal(t, credited, cached.User.Balance, "authentication must see the deposit for every user key")
			}
			assert.NoError(t, billing.CheckBillingEligibility(ctx, user, nil, nil, nil, ""),
				"the first request after recharge must not be rejected for insufficient balance")
			reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
			require.NoError(t, err)
			require.Equal(t, OrderStatusCompleted, reloaded.Status)
			if tt.recovered {
				require.Empty(t, redeemRepo.useCalls, "recovery must not credit a redeemed code again")
			}
		})
	}
}

func TestRedeemBalanceInvalidationSurvivesRequestCancellation(t *testing.T) {
	cache := &authCacheStub{}
	repo := &authRepoStub{listKeysByUserID: func(ctx context.Context, userID int64) ([]string, error) {
		return []string{"recharge-key"}, ctx.Err()
	}}
	auth := NewAPIKeyService(repo, nil, nil, nil, nil, cache, &config.Config{})
	redeem := &RedeemService{authCacheInvalidator: auth}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	redeem.invalidateRedeemCaches(ctx, 42, &RedeemCode{Type: RedeemTypeBalance})

	require.Len(t, cache.deleteAuthKeys, 1, "a committed deposit must invalidate auth even if its request was cancelled")
}

type rechargeBlockingBalanceCache struct {
	billingCacheWorkerStub
	started chan struct{}
	release chan struct{}
}

func (c *rechargeBlockingBalanceCache) InvalidateUserBalance(ctx context.Context, _ int64) error {
	close(c.started)
	select {
	case <-c.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestRedeemBalanceInvalidationWaitsForBalanceCache(t *testing.T) {
	cache := &rechargeBlockingBalanceCache{started: make(chan struct{}), release: make(chan struct{})}
	billing := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(billing.Stop)
	redeem := &RedeemService{billingCacheService: billing}
	returned := make(chan struct{})
	t.Cleanup(func() {
		close(cache.release)
		select {
		case <-returned:
		case <-time.After(2 * time.Second):
			t.Error("balance cache invalidation did not finish after release")
		}
	})
	go func() {
		redeem.invalidateRedeemCaches(context.Background(), 42, &RedeemCode{Type: RedeemTypeBalance})
		close(returned)
	}()

	select {
	case <-cache.started:
	case <-time.After(2 * time.Second):
		t.Fatal("balance cache invalidation did not start")
	}
	select {
	case <-returned:
		t.Fatal("recharge returned while the old balance was still cached")
	case <-time.After(50 * time.Millisecond):
	}
}

func paymentCacheTestBonusWallet(t *testing.T, order *dbent.PaymentOrder, recovered bool) *BonusWallet {
	t.Helper()
	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, database)))
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		_ = client.Close()
	})
	orderID := strconv.FormatInt(order.ID, 10)
	mock.ExpectBegin()
	currentBalance := "0"
	if recovered {
		currentBalance = "90"
	}
	mock.ExpectQuery(`SELECT balance::text FROM users`).WithArgs(order.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(currentBalance))
	grantRows := sqlmock.NewRows([]string{"id"})
	if !recovered {
		grantRows.AddRow(7)
	}
	mock.ExpectQuery(`INSERT INTO wallet_bonus_grants`).
		WithArgs(order.UserID, bonusGrantSourcePaymentOrder, orderID, "10", sqlmock.AnyArg()).
		WillReturnRows(grantRows)
	if recovered {
		mock.ExpectQuery(`SELECT id, expires_at FROM wallet_bonus_grants`).
			WithArgs(bonusGrantSourcePaymentOrder, orderID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at"}).AddRow(7, time.Now().Add(30*24*time.Hour)))
	} else {
		mock.ExpectQuery(`UPDATE users SET balance = balance \+ \$1`).WithArgs("90", order.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("90"))
		mock.ExpectExec(`INSERT INTO wallet_bonus_transactions`).
			WithArgs(order.UserID, int64(7), "10", "payment_order:"+orderID+":fulfill", bonusGrantSourcePaymentOrder, orderID, "90").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`UPDATE payment_orders SET bonus_expires_at`).
			WithArgs(sqlmock.AnyArg(), order.ID).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
	return NewBonusWallet(client)
}

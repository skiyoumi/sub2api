package service

import (
	"context"
	"regexp"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestBonusWalletExpireBatchWritesNumericBalanceAfter(t *testing.T) {
	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, database)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT id, user_id, remaining_amount::text, source_type, source_id FROM wallet_bonus_grants`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "remaining_amount", "source_type", "source_id"}).
			AddRow(7, 42, "10.5", bonusGrantSourcePaymentOrder, "99"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT balance::text FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("100.5"))
	mock.ExpectExec(`UPDATE users SET balance = balance - \$1`).
		WithArgs("10.5", int64(42)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE wallet_bonus_grants SET remaining_amount = 0, status = 'EXPIRED'`).
		WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO wallet_bonus_transactions.*\(SELECT balance FROM users WHERE id = \$1\)`).
		WithArgs(int64(42), int64(7), "10.5", "expiry:7", bonusGrantSourcePaymentOrder, "99").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewBonusWallet(client).ExpireBatch(context.Background(), 500)
	require.NoError(t, err)
	require.Equal(t, 1, result.Grants)
	require.Equal(t, "10.5", result.Amount.String())
	require.NoError(t, mock.ExpectationsWereMet())
}
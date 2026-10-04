package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRechargeBonusPackageCompatibilityMigration(t *testing.T) {
	content, err := FS.ReadFile("242_recharge_bonus_tiers_package_compat.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS payment_orders_bonus_shape")
	require.Contains(t, sql, "bonus_amount > 0 AND bonus_validity_days > 0")
	require.Contains(t, sql, "bonus_amount > 0 AND recharge_package_id IS NULL AND bonus_validity_days = 0 AND bonus_expires_at IS NULL")
}

package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBuildBonusSpendPlanUsesFEFOOrder(t *testing.T) {
	now := time.Now()
	plan := BuildBonusSpendPlan(decimal.RequireFromString("8.5"), []BonusSpendGrant{
		{GrantID: 10, ExpiresAt: now.Add(time.Hour), Available: decimal.RequireFromString("2")},
		{GrantID: 20, ExpiresAt: now.Add(2 * time.Hour), Available: decimal.RequireFromString("3.5")},
		{GrantID: 30, ExpiresAt: now.Add(3 * time.Hour), Available: decimal.RequireFromString("10")},
	})

	require.Equal(t, "8.5", plan.Bonus.String())
	require.Equal(t, "0", plan.Permanent.String())
	require.Equal(t, []int64{10, 20, 30}, []int64{plan.Allocations[0].GrantID, plan.Allocations[1].GrantID, plan.Allocations[2].GrantID})
	require.Equal(t, "3", plan.Allocations[2].Amount.String())
}

func TestBuildBonusSpendPlanFallsBackToPermanentBalance(t *testing.T) {
	plan := BuildBonusSpendPlan(decimal.RequireFromString("5"), []BonusSpendGrant{
		{GrantID: 1, Available: decimal.RequireFromString("1.25")},
	})
	require.Equal(t, "1.25", plan.Bonus.String())
	require.Equal(t, "3.75", plan.Permanent.String())
}

func TestCalculateBonusRefundTargetUsesCreditedRatio(t *testing.T) {
	target := CalculateBonusRefundTarget(
		decimal.RequireFromString("33"),
		decimal.RequireFromString("3"),
		decimal.RequireFromString("11"),
	)
	require.True(t, target.Equal(decimal.RequireFromString("1")))
}

func TestCalculateBonusRefundTargetCapsAtFullBonus(t *testing.T) {
	target := CalculateBonusRefundTarget(
		decimal.RequireFromString("10"),
		decimal.RequireFromString("2"),
		decimal.RequireFromString("12"),
	)
	require.True(t, target.Equal(decimal.RequireFromString("2")))
}

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRechargePackagesNormalizesAndSorts(t *testing.T) {
	packages, err := ParseRechargePackages(`[
		{"id":"pkg_b","amount":"30","bonus_amount":"1.25000000","bonus_validity_days":7,"recommended":true,"enabled":true,"sort_order":20},
		{"id":"pkg_a","amount":"10.0","bonus_amount":"0","bonus_validity_days":0,"recommended":false,"enabled":true,"sort_order":10}
	]`, true, 1, 100)
	require.NoError(t, err)
	require.Equal(t, "pkg_a", packages[0].ID)
	require.Equal(t, "10.00", packages[0].Amount)
	require.Equal(t, "1.25", packages[1].BonusAmount)
}

func TestParseRechargePackagesAcceptsNumericAmountsFromAdminForm(t *testing.T) {
	packages, err := ParseRechargePackages(`[
		{"id":"pkg_30","amount":30,"bonus_amount":1.25,"bonus_validity_days":7,"recommended":true,"enabled":true,"sort_order":10}
	]`, true, 1, 100)
	require.NoError(t, err)
	require.Equal(t, "30.00", packages[0].Amount)
	require.Equal(t, "1.25", packages[0].BonusAmount)
}

func TestParseRechargePackagesRejectsInvalidConfigurations(t *testing.T) {
	tests := map[string]string{
		"unknown field": `[ {"id":"pkg_a","amount":"10","bonus_amount":"0","bonus_validity_days":0,"enabled":true,"sort_order":1,"typo":true} ]`,
		"duplicate enabled amount": `[
			{"id":"pkg_a","amount":"10","bonus_amount":"0","bonus_validity_days":0,"enabled":true,"sort_order":1},
			{"id":"pkg_b","amount":"10.00","bonus_amount":"0","bonus_validity_days":0,"enabled":true,"sort_order":2}
		]`,
		"bonus without validity":    `[ {"id":"pkg_a","amount":"10","bonus_amount":"1","bonus_validity_days":0,"enabled":true,"sort_order":1} ]`,
		"too much amount precision": `[ {"id":"pkg_a","amount":"10.001","bonus_amount":"0","bonus_validity_days":0,"enabled":true,"sort_order":1} ]`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseRechargePackages(raw, true, 1, 100)
			require.Error(t, err)
		})
	}
}

func TestParseRechargePackagesKeepsLegacyDefaults(t *testing.T) {
	packages, err := ParseRechargePackages("", false, 1, 0)
	require.NoError(t, err)
	require.Empty(t, packages)
}

func TestParsePaymentConfigDefaultsToCustomRecharge(t *testing.T) {
	cfg := (&PaymentConfigService{}).parsePaymentConfig(map[string]string{})
	require.False(t, cfg.RechargePackagesEnabled)
	require.True(t, cfg.AllowCustomRecharge)
}

func TestResolveRechargePackageUsesServerAmounts(t *testing.T) {
	cfg := &PaymentConfig{
		RechargePackagesEnabled:   true,
		AllowCustomRecharge:       true,
		BalanceRechargeMultiplier: 1.2,
		RechargePackages: []RechargePackage{{
			ID: "pkg_30", Amount: "30.00", BonusAmount: "1.50",
			BonusValidityDays: 7, Enabled: true,
		}},
	}
	selection, err := ResolveRechargePackage(cfg, "pkg_30", 30)
	require.NoError(t, err)
	require.Equal(t, "36", selection.PermanentAmount.String())
	require.Equal(t, "1.5", selection.BonusAmount.String())
	require.Equal(t, "37.5", selection.CreditedAmount.String())
	require.NotEmpty(t, selection.ConfigHash)

	restored, err := RechargePackageSelectionFromProviderSnapshot(map[string]any{"_recharge_package": selection.Snapshot()})
	require.NoError(t, err)
	require.Equal(t, selection.CreditedAmount.String(), restored.CreditedAmount.String())
}

func TestResolveRechargePackageRejectsClientAndConfigurationDrift(t *testing.T) {
	cfg := &PaymentConfig{
		RechargePackagesEnabled:   true,
		AllowCustomRecharge:       false,
		BalanceRechargeMultiplier: 1,
		RechargePackages:          []RechargePackage{{ID: "pkg_10", Amount: "10.00", BonusAmount: "1", BonusValidityDays: 1, Enabled: true}},
	}
	_, err := ResolveRechargePackage(cfg, "", 10)
	require.ErrorContains(t, err, "custom recharge amount is disabled")
	_, err = ResolveRechargePackage(cfg, "pkg_10", 9)
	require.ErrorContains(t, err, "recharge package has changed")
	_, err = ResolveRechargePackage(cfg, "missing", 10)
	require.ErrorContains(t, err, "recharge package not found")
}

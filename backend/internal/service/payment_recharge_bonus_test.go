//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateOrderRejectsPackageHashFromBeforeBonusExclusion(t *testing.T) {
	ctx := context.Background()
	config := &PaymentConfigService{settingRepo: &paymentConfigSettingRepoStub{values: map[string]string{
		SettingPaymentEnabled: "true", SettingRechargePackagesEnabled: "true",
		SettingRechargePackages: `[{"id":"pkg_30","amount":"30.00","bonus_amount":"5","bonus_validity_days":7,"enabled":true}]`,
	}}}
	cfg, err := config.GetPaymentConfig(ctx)
	require.NoError(t, err)
	oldOffer, err := ResolveRechargePackage(cfg, &User{}, "pkg_30", 30)
	require.NoError(t, err)
	svc := &PaymentService{
		configService: config,
		userRepo:      &rechargeBonusUserRepo{user: User{ID: 42, Status: StatusActive, RechargeBonusDisabled: true}},
	}
	_, err = svc.CreateOrder(ctx, CreateOrderRequest{UserID: 42, Amount: 30, PaymentType: "alipay", RechargePackageID: "pkg_30", RechargePackageHash: oldOffer.ConfigHash})
	require.ErrorContains(t, err, "recharge package has changed")
}

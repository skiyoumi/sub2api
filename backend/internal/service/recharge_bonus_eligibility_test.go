package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type rechargeBonusUserRepo struct {
	UserRepository
	user   User
	err    error
	fields UserUpdateFields
}

func (r *rechargeBonusUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	if r.err != nil {
		return nil, r.err
	}
	if id != r.user.ID {
		return nil, ErrUserNotFound
	}
	copy := r.user
	return &copy, nil
}

func (r *rechargeBonusUserRepo) Update(_ context.Context, user *User, fields UserUpdateFields) error {
	r.fields = fields
	if fields.RechargeBonusDisabled {
		r.user.RechargeBonusDisabled = user.RechargeBonusDisabled
	}
	return nil
}

func TestAdminUpdateUserRechargeBonusEligibility(t *testing.T) {
	repo := &rechargeBonusUserRepo{user: User{ID: 42, Balance: 125}}
	svc := &adminServiceImpl{userRepo: repo}
	ctx := context.Background()
	for _, disabled := range []bool{true, false} {
		updated, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{RechargeBonusDisabled: &disabled})
		require.NoError(t, err)
		require.Equal(t, disabled, updated.RechargeBonusDisabled)
		require.Equal(t, UserUpdateFields{RechargeBonusDisabled: true}, repo.fields)
		require.Equal(t, float64(125), repo.user.Balance)
	}
	repo.user.RechargeBonusDisabled = true
	_, err := svc.UpdateUser(ctx, 42, &UpdateUserInput{})
	require.NoError(t, err)
	require.True(t, repo.user.RechargeBonusDisabled, "omitting the flag must preserve eligibility")
	require.True(t, repo.fields.IsEmpty())
}

func TestCheckoutRechargePackagesUseCurrentUserEligibility(t *testing.T) {
	repo := &rechargeBonusUserRepo{user: User{ID: 42, RechargeBonusDisabled: true}}
	svc := &PaymentService{userRepo: repo}
	packages := []RechargePackage{{ID: "pkg_30", Amount: "30", BonusAmount: "5", BonusValidityDays: 7, Enabled: true}}
	ctx := context.Background()
	actual, err := svc.GetRechargePackagesForUser(ctx, 42, packages)
	require.NoError(t, err)
	require.Equal(t, "0", actual[0].BonusAmount)
	require.Zero(t, actual[0].BonusValidityDays)
	require.Equal(t, "30", actual[0].Amount)
	require.Equal(t, "5", packages[0].BonusAmount)

	repo.user.RechargeBonusDisabled = false
	actual, err = svc.GetRechargePackagesForUser(ctx, 42, packages)
	require.NoError(t, err)
	require.Equal(t, packages, actual, "re-enabling bonuses takes effect without cached eligibility")

	repo.err = errors.New("user lookup unavailable")
	actual, err = svc.GetRechargePackagesForUser(ctx, 42, packages)
	require.ErrorIs(t, err, repo.err)
	require.Nil(t, actual, "failed eligibility lookup must not expose a bonus offer")
}

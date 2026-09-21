package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryRechargeBonusEligibility(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()
	u := &service.User{Email: "bonus@example.test", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive, Balance: 125}
	require.NoError(t, repo.Create(ctx, u))
	got, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.False(t, got.RechargeBonusDisabled)

	for _, disabled := range []bool{true, false} {
		u.RechargeBonusDisabled = disabled
		require.NoError(t, repo.Update(ctx, u, service.UserUpdateFields{RechargeBonusDisabled: true}))
		got, err = repo.GetByID(ctx, u.ID)
		require.NoError(t, err)
		require.Equal(t, disabled, got.RechargeBonusDisabled)
		require.Equal(t, float64(125), got.Balance)
	}
	// An unrelated stale profile edit must not overwrite a concurrently changed flag.
	u.RechargeBonusDisabled = true
	require.NoError(t, repo.Update(ctx, u, service.UserUpdateFields{RechargeBonusDisabled: true}))
	got.Username = "renamed"
	require.NoError(t, repo.Update(ctx, got, service.UserUpdateFields{Username: true}))
	got, err = repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	require.True(t, got.RechargeBonusDisabled)
}

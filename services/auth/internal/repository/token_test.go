//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

func TestSaveRefreshToken(t *testing.T) {
	client := valkeyContainer.Client(t)
	repo := repository.NewTokenRepository(client)
	ctx := context.Background()

	userID := testutil.RandomUUID()
	token := testutil.RandomUUID().String()

	t.Run("success", func(t *testing.T) {
		err := repo.SaveRefreshToken(ctx, userID, token, time.Minute)
		require.NoError(t, err)

		// Verify the token is retrievable.
		gotID, err := repo.GetUserIDByRefreshToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, userID, gotID)
	})

	t.Run("overwrite existing token", func(t *testing.T) {
		newUserID := testutil.RandomUUID()
		err := repo.SaveRefreshToken(ctx, newUserID, token, time.Minute)
		require.NoError(t, err)

		gotID, err := repo.GetUserIDByRefreshToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, newUserID, gotID)
	})
}

func TestGetUserIDByRefreshToken(t *testing.T) {
	client := valkeyContainer.Client(t)
	repo := repository.NewTokenRepository(client)
	ctx := context.Background()

	userID := testutil.RandomUUID()
	token := testutil.RandomUUID().String()

	require.NoError(t, repo.SaveRefreshToken(ctx, userID, token, time.Minute))

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:  "found",
			token: token,
		},
		{
			name:    "not found returns ErrTokenNotFound",
			token:   testutil.RandomUUID().String(),
			wantErr: repository.ErrTokenNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, err := repo.GetUserIDByRefreshToken(ctx, tc.token)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, gotID)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, userID, gotID)
		})
	}
}

func TestDeleteRefreshToken(t *testing.T) {
	client := valkeyContainer.Client(t)
	repo := repository.NewTokenRepository(client)
	ctx := context.Background()

	userID := testutil.RandomUUID()
	token := testutil.RandomUUID().String()

	require.NoError(t, repo.SaveRefreshToken(ctx, userID, token, time.Minute))

	t.Run("success", func(t *testing.T) {
		err := repo.DeleteRefreshToken(ctx, token)
		require.NoError(t, err)

		_, err = repo.GetUserIDByRefreshToken(ctx, token)
		require.ErrorIs(t, err, repository.ErrTokenNotFound)
	})

	t.Run("already deleted is idempotent", func(t *testing.T) {
		// Token was already deleted above — deleting again should not error.
		err := repo.DeleteRefreshToken(ctx, token)
		require.NoError(t, err)
	})
}

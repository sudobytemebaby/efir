//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

func TestCreateAccount(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewAccountRepository(pool)
	ctx := context.Background()

	tests := []struct {
		name      string
		email     string
		wantErr   bool
		errTarget error
	}{
		{
			name:  "success",
			email: testutil.RandomEmail(),
		},
		{
			name:      "duplicate email returns error",
			email:     "", // set below after first insert
			wantErr:   true,
		},
	}

	// Pre-create an account for the duplicate test.
	dupEmail := testutil.RandomEmail()
	_, err := repo.CreateAccount(ctx, dupEmail, "hash")
	require.NoError(t, err)
	tests[1].email = dupEmail

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			acc, err := repo.CreateAccount(ctx, tc.email, "hash")
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, acc)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, acc)
			assert.Equal(t, tc.email, acc.Email)
			assert.NotEmpty(t, acc.ID)
			assert.NotEmpty(t, acc.PasswordHash)
		})
	}
}

func TestGetAccountByEmail(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewAccountRepository(pool)
	ctx := context.Background()

	email := testutil.RandomEmail()
	created, err := repo.CreateAccount(ctx, email, "hashval")
	require.NoError(t, err)

	tests := []struct {
		name      string
		email     string
		wantErr   error
	}{
		{
			name:  "found",
			email: email,
		},
		{
			name:    "not found returns ErrAccountNotFound",
			email:   testutil.RandomEmail(),
			wantErr: repository.ErrAccountNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			acc, err := repo.GetAccountByEmail(ctx, tc.email)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, acc)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, created.ID, acc.ID)
			assert.Equal(t, email, acc.Email)
		})
	}
}

func TestGetAccountByID(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewAccountRepository(pool)
	ctx := context.Background()

	email := testutil.RandomEmail()
	created, err := repo.CreateAccount(ctx, email, "hashval")
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func() (interface{}, error)
		wantErr error
	}{
		{
			name:    "found",
		},
		{
			name:    "not found returns ErrAccountNotFound",
			wantErr: repository.ErrAccountNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := created.ID
			if tc.wantErr != nil {
				id = testutil.RandomUUID()
			}
			acc, err := repo.GetAccountByID(ctx, id)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, acc)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, created.ID, acc.ID)
			assert.Equal(t, email, acc.Email)
		})
	}
}

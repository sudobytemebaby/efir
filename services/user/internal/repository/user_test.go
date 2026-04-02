//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
)

var pgContainer *testutil.PostgresContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	pgContainer = testutil.NewPostgresContainer(ctx, "../../migrations")
	exitCode := m.Run()
	_ = pgContainer.Terminate(ctx)
	os.Exit(exitCode)
}

func createUser(t *testing.T, repo repository.UserRepository) *repository.User {
	t.Helper()
	user, err := repo.CreateUser(context.Background(),
		testutil.RandomUUID(),
		testutil.RandomUsername(),
		"Display Name",
	)
	require.NoError(t, err)
	return user
}

func TestCreateUser(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		id := testutil.RandomUUID()
		user, err := repo.CreateUser(ctx, id, testutil.RandomUsername(), "Test User")
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, id, user.ID)
		assert.Equal(t, "Test User", user.DisplayName)
	})

	t.Run("duplicate id returns ErrUserAlreadyExists", func(t *testing.T) {
		existing := createUser(t, repo)
		_, err := repo.CreateUser(ctx, existing.ID, testutil.RandomUsername(), "Another Name")
		require.ErrorIs(t, err, repository.ErrUserAlreadyExists)
	})
}

func TestGetUserByID(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	existing := createUser(t, repo)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{name: "found", id: existing.ID},
		{name: "not found", id: testutil.RandomUUID(), wantErr: repository.ErrUserNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := repo.GetUserByID(ctx, tc.id)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, user)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, existing.ID, user.ID)
			assert.Equal(t, existing.Username, user.Username)
		})
	}
}

func TestGetUsersByIDs(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	u1 := createUser(t, repo)
	u2 := createUser(t, repo)
	u3 := createUser(t, repo)

	tests := []struct {
		name      string
		ids       []uuid.UUID
		wantCount int
	}{
		{
			name:      "all found",
			ids:       []uuid.UUID{u1.ID, u2.ID, u3.ID},
			wantCount: 3,
		},
		{
			name:      "partial match",
			ids:       []uuid.UUID{u1.ID, testutil.RandomUUID()},
			wantCount: 1,
		},
		{
			name:      "none found returns empty",
			ids:       []uuid.UUID{testutil.RandomUUID()},
			wantCount: 0,
		},
		{
			name:      "nil input returns nil",
			ids:       nil,
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users, err := repo.GetUsersByIDs(ctx, tc.ids)
			require.NoError(t, err)
			assert.Len(t, users, tc.wantCount)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	strPtr := func(s string) *string { return &s }

	createUserWithFields := func(t *testing.T) *repository.User {
		t.Helper()
		user, err := repo.CreateUser(context.Background(),
			testutil.RandomUUID(),
			testutil.RandomUsername(),
			"Display Name",
		)
		require.NoError(t, err)

		updated, err := repo.UpdateUser(context.Background(), user.ID, nil, strPtr("https://example.com/avatar.jpg"), strPtr("My bio"))
		require.NoError(t, err)
		return updated
	}

	tests := []struct {
		name        string
		id          func() uuid.UUID
		displayName *string
		avatarURL   *string
		bio         *string
		wantErr     error
		checkFn     func(t *testing.T, u *repository.User)
	}{
		{
			name:        "update display_name only",
			id:          func() uuid.UUID { return createUser(t, repo).ID },
			displayName: strPtr("New Name"),
			checkFn: func(t *testing.T, u *repository.User) {
				t.Helper()
				assert.Equal(t, "New Name", u.DisplayName)
			},
		},
		{
			name:        "update all fields",
			id:          func() uuid.UUID { return createUser(t, repo).ID },
			displayName: strPtr("All Fields"),
			avatarURL:   strPtr("https://example.com/avatar.jpg"),
			bio:         strPtr("My bio"),
			checkFn: func(t *testing.T, u *repository.User) {
				t.Helper()
				assert.Equal(t, "All Fields", u.DisplayName)
				require.NotNil(t, u.AvatarURL)
				assert.Equal(t, "https://example.com/avatar.jpg", *u.AvatarURL)
				require.NotNil(t, u.Bio)
				assert.Equal(t, "My bio", *u.Bio)
			},
		},
		{
			name:        "clear avatar_url to NULL",
			id:          func() uuid.UUID { return createUserWithFields(t).ID },
			displayName: nil,
			avatarURL:   strPtr(""),
			checkFn: func(t *testing.T, u *repository.User) {
				t.Helper()
				assert.Equal(t, "Display Name", u.DisplayName)
				require.Nil(t, u.AvatarURL, "avatar_url should be NULL")
				require.NotNil(t, u.Bio)
			},
		},
		{
			name:        "clear bio to NULL",
			id:          func() uuid.UUID { return createUserWithFields(t).ID },
			displayName: nil,
			bio:         strPtr(""),
			checkFn: func(t *testing.T, u *repository.User) {
				t.Helper()
				assert.Equal(t, "Display Name", u.DisplayName)
				require.NotNil(t, u.AvatarURL)
				require.Nil(t, u.Bio, "bio should be NULL")
			},
		},
		{
			name:    "not found returns ErrUserNotFound",
			id:      func() uuid.UUID { return testutil.RandomUUID() },
			wantErr: repository.ErrUserNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := repo.UpdateUser(ctx, tc.id(), tc.displayName, tc.avatarURL, tc.bio)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, user)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, user)
			if tc.checkFn != nil {
				tc.checkFn(t, user)
			}
		})
	}
}

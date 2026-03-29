package testutil

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// RandomUUID returns a new random UUID.
func RandomUUID() uuid.UUID {
	return uuid.New()
}

// RandomEmail returns a unique test email address.
func RandomEmail() string {
	return fmt.Sprintf("user-%s@test.example", uuid.New().String()[:8])
}

// RandomUsername returns a unique username safe for database unique constraints.
func RandomUsername() string {
	return "user-" + uuid.New().String()[:8]
}

// RandomRoomName returns a unique room name.
func RandomRoomName() string {
	return "room-" + uuid.New().String()[:8]
}

// HashedPassword returns a bcrypt hash of password using MinCost for speed.
// Use this when pre-creating accounts in test fixtures, not in production.
func HashedPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(hash)
}

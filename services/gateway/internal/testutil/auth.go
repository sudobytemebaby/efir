package testutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TestSecret = "test-secret"

func AuthHeader(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(TestSecret))
	return "Bearer " + signed
}

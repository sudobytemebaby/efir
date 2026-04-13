package testutil

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TestSecret = "test-secret"

func AccessToken(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(TestSecret))
	return signed
}

func SetAccessCookie(req *http.Request, userID string) {
	req.AddCookie(&http.Cookie{Name: "access_token", Value: AccessToken(userID)})
}

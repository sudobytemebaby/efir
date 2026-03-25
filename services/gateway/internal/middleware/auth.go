package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type (
	contextKeyUserID    struct{}
	contextKeyRequestID struct{}
)

const (
	HeaderRequestID      = "X-Request-ID"
	MetadataKeyRequestID = "x-request-id"
)

func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(HeaderRequestID)
			if requestID == "" {
				requestID = uuid.New().String()
			}
			w.Header().Set(HeaderRequestID, requestID)

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			sub, ok := claims["sub"].(string)
			if !ok || sub == "" {
				http.Error(w, "missing sub claim", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID{}, sub)
			ctx = context.WithValue(ctx, contextKeyRequestID{}, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(contextKeyUserID{}).(string)
	return userID, ok
}

// MustGetUserID returns the user ID from context.
// Safe to call in handlers behind the JWTAuth middleware.
func MustGetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(contextKeyUserID{}).(string)
	return userID
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(contextKeyRequestID{}).(string)
	return requestID, ok
}

func InjectRequestIDToOutgoingContext(ctx context.Context) context.Context {
	requestID, ok := GetRequestID(ctx)
	if !ok || requestID == "" {
		return ctx
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	md.Set(MetadataKeyRequestID, requestID)
	return metadata.NewOutgoingContext(ctx, md)
}

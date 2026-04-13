package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
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

			cookie, err := r.Cookie("access_token")
			if err != nil {
				handler.WriteCode(w, errors.CodeUnauthenticated)
				return
			}

			tokenString := cookie.Value

			token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				handler.WriteCode(w, errors.CodeUnauthenticated)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				handler.WriteCode(w, errors.CodeUnauthenticated)
				return
			}

			sub, ok := claims["sub"].(string)
			if !ok || sub == "" {
				handler.WriteCode(w, errors.CodeUnauthenticated)
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
	userID, ok := ctx.Value(contextKeyUserID{}).(string)
	if !ok {
		panic("user ID not found in context")
	}
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

package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

func UserRateLimiter(client vk.Client, requests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r.Context())
			if !ok {
				handler.WriteCode(w, errors.CodeUnauthenticated)
				return
			}

			ttlSeconds := strconv.Itoa(int(window.Seconds()))
			key := valkey.GatewayRateLimitKey("user", userID, ttlSeconds)

			result, err := client.Do(r.Context(), client.B().Eval().Script(valkey.IncrWithExpiryScript).Numkeys(1).Key(key).Arg(ttlSeconds).Build()).ToInt64()
			if err != nil {
				handler.WriteCode(w, errors.CodeInternal)
				return
			}

			if result > int64(requests) {
				handler.WriteCode(w, errors.CodeRateLimited)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

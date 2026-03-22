package middleware

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

// IPRateLimiter implements rate limiting by client IP address.
// The window parameter is included in the key for proper rate limiting per time window.
// TODO: Consider using sliding window log instead of fixed window with window in key
// to avoid edge cases when RATE_LIMIT_WINDOW changes.
func IPRateLimiter(client vk.Client, requests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			ttlSeconds := strconv.Itoa(int(window.Seconds()))
			key := valkey.GatewayRateLimitKey("ip", ip, ttlSeconds)

			result, err := client.Do(r.Context(), client.B().Eval().Script(valkey.IncrWithExpiryScript).Numkeys(1).Key(key).Arg(ttlSeconds).Build()).ToInt64()
			if err != nil {
				http.Error(w, "rate limit check failed", http.StatusInternalServerError)
				return
			}

			if result > int64(requests) {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func UserRateLimiter(client vk.Client, requests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ttlSeconds := strconv.Itoa(int(window.Seconds()))
			key := valkey.GatewayRateLimitKey("user", userID, ttlSeconds)

			result, err := client.Do(r.Context(), client.B().Eval().Script(valkey.IncrWithExpiryScript).Numkeys(1).Key(key).Arg(ttlSeconds).Build()).ToInt64()
			if err != nil {
				http.Error(w, "rate limit check failed", http.StatusInternalServerError)
				return
			}

			if result > int64(requests) {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

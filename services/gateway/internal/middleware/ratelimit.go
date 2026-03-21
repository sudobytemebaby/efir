package middleware

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

const luaScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`

func IPRateLimiter(client vk.Client, requests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			ttlSeconds := strconv.Itoa(int(window.Seconds()))
			key := valkey.GatewayRateLimitKey("ip", ip, ttlSeconds)

			result, err := client.Do(r.Context(), client.B().Eval().Script(luaScript).Numkeys(1).Key(key).Arg(ttlSeconds).Build()).ToInt64()
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

			result, err := client.Do(r.Context(), client.B().Eval().Script(luaScript).Numkeys(1).Key(key).Arg(ttlSeconds).Build()).ToInt64()
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

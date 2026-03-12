// Package valkey provides constants and helpers for Valkey key naming.
package valkey

const (
	//nolint:gosec
	AuthRefreshToken = "auth:refresh:%s"
	GatewayRateLimit = "gateway:ratelimit:%s:%s:%s"
	WSPubsubChannel  = "ws:pubsub:%s"
	PresenceOnline   = "presence:online:%s"
)

func AuthRefreshKey(token string) string {
	return "auth:refresh:" + token
}

func GatewayRateLimitKey(limitType, value, window string) string {
	return "gateway:ratelimit:" + limitType + ":" + value + ":" + window
}

func WSPubsubChannelKey(channel string) string {
	return "ws:pubsub:" + channel
}

func PresenceOnlineKey(userID string) string {
	return "presence:online:" + userID
}

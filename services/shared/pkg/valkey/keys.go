// Package valkey provides constants and helpers for Valkey key naming.
package valkey

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

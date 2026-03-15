package valkey

func AuthRefreshKey(token string) string {
	return "auth:refresh:" + token
}

func AuthRateLimitKey(action, email string) string {
	return "auth:ratelimit:" + action + ":" + email
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

# valkey

Valkey key patterns and Lua scripts for caching, rate limiting, and presence tracking.

## Key Patterns

All keys follow a `prefix:subtype:value` naming convention.

### Auth Keys

| Function                          | Pattern                           | Purpose                      |
| --------------------------------- | --------------------------------- | ---------------------------- |
| `AuthRefreshKey(token)`           | `auth:refresh:{token}`            | Refresh token store          |
| `AuthRateLimitKey(action, email)` | `auth:ratelimit:{action}:{email}` | Login/register rate limiting |

### Gateway Keys

| Function                                        | Pattern                                     | Purpose                          |
| ----------------------------------------------- | ------------------------------------------- | -------------------------------- |
| `GatewayRateLimitKey(limitType, value, window)` | `gateway:ratelimit:{type}:{value}:{window}` | Per-user/IP rate limiting        |
| `GatewayWSTicketKey(ticket)`                    | `gateway:ws:ticket:{ticket}`                | WebSocket ticket temporary store |

### WebSocket Keys

| Function                      | Pattern               | Purpose                 |
| ----------------------------- | --------------------- | ----------------------- |
| `WSPubsubChannelKey(channel)` | `ws:pubsub:{channel}` | Pub/sub channel mapping |

### Presence Keys

| Function                    | Pattern                    | Purpose           |
| --------------------------- | -------------------------- | ----------------- |
| `PresenceOnlineKey(userID)` | `presence:online:{userID}` | Online status TTL |

## Lua Scripts

### `IncrWithExpiryScript`

Atomic increment with expiry. Used for sliding-window rate limiting.

```lua
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
```

- `KEYS[1]` - the key to increment
- `ARGV[1]` - TTL in seconds
- Returns current count after increment

## Usage

Services use the valkey-go client to interact with Valkey:

```go
import vk "github.com/valkey-io/valkey-go"

// Connect
client, err := vk.NewClient(vk.ClientOption{
    InitAddress: []string{"localhost:6379"},
})

// Rate limiting with expiry
result, err := client.Do(ctx, client.B().Eval().
    Script(sharedvalkey.IncrWithExpiryScript).
    Keys("gateway:ratelimit:user:abc123:minute").
    Arg("60"). // 60 second window
    Build()).AsInt64()

if result > 100 {
    return errors.CodeRateLimited.Error("rate limit exceeded")
}

// Generate a WS ticket
ticket := uuid.New().String()
client.Do(ctx, client.B().Set().
    Key(sharedvalkey.GatewayWSTicketKey(ticket)).
    Value(userID).
    Exat(time.Now().Add(5*time.Minute).Unix()).
    Build())
```

package valkey

// IncrWithExpiryScript atomically increments a counter and sets its TTL only on the
// first increment, so the expiry window always starts at the first request in the period.
const IncrWithExpiryScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`

// GetAndDeleteScript atomically reads and deletes a key in one round-trip,
// used to make tokens single-use (e.g. refresh tokens, WS tickets).
const GetAndDeleteScript = `
local value = redis.call('GET', KEYS[1])
if value then
    redis.call('DEL', KEYS[1])
end
return value
`

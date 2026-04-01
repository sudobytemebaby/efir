## ADR-015: Random Username Generation

## Status

Accepted

## Context

The User Service originally generated usernames from email prefixes (e.g., `john@example.com` → `john`). This caused username collisions when multiple users had the same local part from different domains (e.g., `alice@gmail.com` and `alice@yahoo.com` both resulted in `alice`).

The PostgreSQL `ON CONFLICT (id) DO NOTHING` only handled ID conflicts, not username conflicts. When a username collision occurred, the unique constraint violation surfaced as an unhandled error, causing the NATS subscriber to Nak and retry forever until `MaxDeliver` was exhausted.

## Decision

Replace email-based username generation with random adjective-noun format (inspired by Docker container names and Battle.net).

### Username Format

Generated usernames follow the pattern: `adjective-noun`

Examples:
- `brave-wolf`
- `calm-duck`
- `swift-eagle`

### Implementation

The `pkg.GenerateUsername()` function in `services/shared/pkg/username.go` generates usernames using cryptographically secure random selection:

```go
func GenerateUsername() (string, error) {
    adjIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(Adjectives))))
    if err != nil {
        return "", err
    }

    nounIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(Nouns))))
    if err != nil {
        return "", err
    }

    return Adjectives[adjIdx.Int64()] + "-" + Nouns[nounIdx.Int64()], nil
}
```

### Collision Handling

The word lists contain:
- 433 adjectives
- 938 nouns
- ~406,000 unique combinations

Even with this large namespace, collisions are possible. The service implements a retry loop:

```go
const maxAttempts = 3

for attempt := 0; attempt < maxAttempts; attempt++ {
    username, err := pkg.GenerateUsername()
    // ... create user ...

    if errors.Is(err, repository.ErrUsernameAlreadyExists) {
        continue  // Retry with new username
    }
}
```

### Repository Changes

Added `ErrUsernameAlreadyExists` error and updated SQL:

```go
const query = `
    INSERT INTO users (id, username, display_name)
    VALUES ($1, $2, $3)
    ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username
    ON CONFLICT (username) DO NOTHING
    RETURNING ...
`
```

Note: `ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username` ensures that if the same user ID is reused, the username is refreshed (e.g., if the username was previously taken, we can assign a new one).

## Alternatives Considered

1. **Email prefix + random suffix**: e.g., `alice-7f3a`. This was rejected in favor of the adjective-noun format as it's more memorable and visually distinctive.

2. **UUID-based usernames**: e.g., `user-a1b2c3d4`. This was rejected as it's less user-friendly than adjective-noun names.

3. **User-chosen usernames**: Delaying username generation to user input was out of scope for this fix.

4. **Longer retry attempts**: Using more than 3 attempts is unnecessary given the ~400k namespace.

## Consequences

- **Positive**:
  - No more username collisions from email prefixes
  - Memorable, distinctive usernames
  - Cryptographically secure randomness
  - Graceful retry on the extremely rare collision

- **Negative**:
  - Users no longer have predictable usernames derived from their email
  - Usernames are now random rather than semantic

## References

- [codename library](https://github.com/lucasepe/codename) - Inspiration for adjective-noun format

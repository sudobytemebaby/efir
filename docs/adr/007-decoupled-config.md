# ADR-007: Decoupled Service Configuration

## Status

Accepted

## Context

Initially, the project used a shared configuration package in `services/shared/pkg/config` that defined common environment variables for all services. While this reduced duplication, it created a tight coupling between services. Any change to a common configuration field would require a global update, and services were forced to import fields they didn't need.

## Decision

Remove the shared configuration package and implement a local `internal/config` package within each service. Each service will now define only the environment variables it specifically requires.

## Alternatives

- **Shared Package with Embedding:** Keep the shared package but have services embed only relevant parts (still keeps coupling to the shared module).
- **External Configuration Service:** Use a centralized config management tool like HashiCorp Consul or Spring Cloud Config (too much overhead for the current MVP).

## Consequences

- **Loose Coupling:** Services are now fully autonomous regarding their configuration.
- **Autonomy:** Each service can choose its own configuration library or structure if needed.
- **Slight Duplication:** Common fields like `NATS_URL` or `POSTGRES_DSN` will be duplicated across `internal/config` packages, but this is a deliberate trade-off for better isolation.

---

## Addendum: Environment and Log Level

### Environment Enum

Each service config declares an `Environment` type validated by `cleanenv`:

```go
type Environment string

const (
    EnvDevelopment Environment = "development"
    EnvProduction  Environment = "production"
)

func (e Environment) Validate() error {
    switch e {
    case EnvDevelopment, EnvProduction:
        return nil
    default:
        return fmt.Errorf("invalid environment %q, allowed: development, production", e)
    }
}
```

`cleanenv` calls `Validate()` automatically during `ReadEnv`. If `ENV` is set to an unknown value the service refuses to start with a clear error message. This prevents misconfigured deployments from silently running in the wrong mode.

`Env` is used to gate development-only features — currently gRPC server reflection, which exposes all service methods to unauthenticated clients and should never run in production.

### Log Level

`LOG_LEVEL` is a separate string field (not an enum) because its validation and parsing logic lives in `services/shared/pkg/logger.ParseLevel`. This avoids duplicating the allowed values list in every service config. An invalid log level is non-fatal — the service falls back to `info` and logs a warning.

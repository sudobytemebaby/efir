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

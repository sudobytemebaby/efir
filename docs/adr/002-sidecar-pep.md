## ADR-002: Sidecar PEP (Policy Enforcement Point)

## Status

Accepted

## Context

We need a Policy Enforcement Point (PEP) to validate incoming requests before they reach service business logic. This includes protobuf validation, schema checking, and policy enforcement.

## Decision

Implement sidecar as a Go gRPC reverse proxy that validates protobuf messages.

## Rationale

- **Single toolchain**: Go only, no Rust toolchain required
- **Native gRPC**: Go has first-class gRPC support, seamless integration
- **Protobuf native**: Direct protobuf message validation without translation
- **Deployment simplicity**: Same Docker workflow as services

## Alternatives Considered

- **Istio/Envoy**: Full service mesh solution
  - Pros: Mature, battle-tested
  - Cons: Excessive for our scale, complex configuration, high resource usage
- **OPA (Open Policy Agent)**: General-purpose policy engine
  - Pros: Flexible, declarative policies
  - Cons: Requires learning Rego language, additional component to maintain

- **Rust-based proxy**: e.g., Tonic-based service
  - Pros: Better performance, memory safety
  - Cons: Different toolchain, no native gRPC server reflection, steeper learning curve

## Consequences

- One sidecar container per service
- Validates incoming protobuf messages against schema
- Can enforce rate limiting, authentication checks
- Requires policy configuration per service
- Simpler than full service mesh

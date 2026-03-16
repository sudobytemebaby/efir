# ADR-009: Protobuf Validation with protovalidate

## Status

Accepted

## Context

gRPC handlers need to validate incoming request fields (email format, password length, non-empty tokens). We need to decide where and how this validation lives.

## Decision

Use `buf.build/go/protovalidate` for runtime validation driven by CEL annotations in `.proto` files.

Validation rules are declared directly in the proto contract:

```protobuf
import "buf/validate/validate.proto";

message RegisterRequest {
  string email = 1 [
    (buf.validate.field).string.email = true,
    (buf.validate.field).string.max_len = 255
  ];
  string password = 2 [
    (buf.validate.field).string.min_len = 8,
    (buf.validate.field).string.max_len = 72
  ];
}
```

A single `protovalidate.Validator` is created at handler startup and reused across requests. Each handler calls `h.validator.Validate(req)` before invoking the service layer.

## Rationale

- **Validation lives in the contract**: rules are visible to all consumers of the proto, including the sidecar PEP in Module 2
- **No code generation required**: `protovalidate` reads CEL expressions from proto descriptors at runtime — `buf.gen.yaml` stays clean with only `protoc-gen-go` and `protoc-gen-go-grpc`
- **Single source of truth**: adding a field to a message and forgetting to validate it is harder when validation is co-located with the field definition
- **Sidecar alignment**: the Rust sidecar PEP will enforce the same proto contracts — having validation rules in the descriptor means the sidecar can reuse them directly

## buf Infrastructure

- `buf.yaml` declares `buf.build/bufbuild/protovalidate` as a dependency so buf can resolve `import "buf/validate/validate.proto"` during lint and generate
- `buf.lock` pins the exact commit for reproducible builds
- No remote plugins in `buf.gen.yaml` — validation is purely a runtime concern

## Alternatives Considered

- **Manual validation in handler**: simple, no dependencies. Rejected because rules are scattered across Go code rather than the contract, and the sidecar cannot reuse them.
- **protoc-gen-validate**: older Envoy-era tool that generates `.pb.validate.go` files with a `.Validate()` method on each message. Rejected because it requires an additional code generation step, generates substantial boilerplate, and the ecosystem is moving toward `protovalidate`.
- **Validation in service layer**: would duplicate proto field constraints in Go code. The service layer should enforce business rules (account already exists, invalid credentials), not field format constraints.

## Consequences

- Field validation rules are co-located with field definitions in `.proto` files
- `NewAuthHandler` returns an error (validator initialization can fail) — callers must handle it
- All future services should declare `buf.validate` annotations on their request messages
- The sidecar PEP in Module 2 can enforce the same constraints without duplicating logic

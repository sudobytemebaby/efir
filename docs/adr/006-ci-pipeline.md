## ADR-006: GitHub Actions CI/CD Pipeline

## Status
Accepted

## Context
We need automated validation and build processes for every push and pull request to ensure code quality and catch issues early.

## Decision
Use GitHub Actions with a multi-stage CI pipeline defined in `.github/workflows/ci.yml`.

## Pipeline Stages

### 1. Proto Lint
- Uses `buf` to lint all `.proto` files
- Runs first to catch proto schema issues before generation

### 2. Proto Generate
- Generates Go code from proto files using `buf generate`
- Installs required plugins: `protoc-gen-go`, `protoc-gen-go-grpc`
- Subsequent jobs depend on this (caches generated code)

### 3. Lint
- Runs `golangci-lint` on all Go code
- Uses config: `.golangci.yml`

### 4. Test
- Runs `go test -coverprofile` for all services
- Uploads coverage report as artifact

### 5. Coverage Gate (Future)
- Currently disabled (will activate in Module 1)
- Enforces coverage thresholds: 70% for service layer, 60% for handler layer

### 6. Build
- Builds Docker images for all services
- Handles missing Dockerfiles gracefully (skips non-existent services)

## Workflow Triggers

```yaml
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
```

This ensures every push to main and every PR is validated.

## Implementation Details

### Go Version
- Uses Go 1.23 for all jobs

### buf Version
- Pinned to version 1.32.0 for consistency

### golangci-lint
- Version: v1.61.0
- Config: `.golangci.yml`

### Services Built
- gateway, auth, user, message, room
- (forward-looking: media, presence, notification, search, websocket, sidecar)

## Alternatives Considered
- **Travis CI**: Older, less integrated with GitHub
- **CircleCI**: More complex configuration, requires separate setup
- **GitLab CI**: Not suitable for GitHub repositories

## Consequences
- Every PR and push is automatically validated
- Proto, lint, test, and build issues caught before merge
- Coverage reporting available
- Docker images built for all services
- No manual CI configuration needed

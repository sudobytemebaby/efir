# ADR-016: nginx as Reverse Proxy

## Status

Accepted

## Context

The project initially used Traefik v3.6 as the edge reverse proxy. Traefik was chosen for its Docker-native service discovery via container labels and dynamic configuration. In practice, its configuration was split across three locations (`traefik.yml`, `dynamic/`, and per-service labels in compose files), making it difficult to reason about the full routing setup at a glance. For a project with two upstream services and straightforward routing rules, this overhead outweighed the benefits.

## Decision

Replace Traefik with nginx 1.27 (Alpine). All routing configuration lives in a single file: `infra/nginx/nginx.conf`.

### Routing

| Host | Upstream | Notes |
|------|----------|-------|
| `api.localhost` | `efir-gateway:8080` | CORS, rate limiting |
| `ws.localhost` | `efir-websocket:8081` | Forward auth via `auth_request` |

### CORS

Handled via nginx `map` directive. Allowed origin list is declared in one place. Unknown origins receive an empty `Access-Control-Allow-Origin` header, which browsers reject.

### Rate Limiting

Global IP-based rate limiting via `limit_req_zone`: 100 req/min, burst of 50. This is a network-edge layer on top of the per-user rate limiting already enforced in Gateway.

### Forward Auth (WebSocket)

nginx `auth_request` replaces Traefik's `forwardAuth` middleware. Before proxying a WebSocket connection, nginx issues an internal subrequest to `GET /auth/validate` on the Gateway. If the Gateway returns 200, the `X-User-Id` response header is extracted and forwarded to the WebSocket service.

### Dynamic DNS Resolution

Docker container names are resolved at request time (not at nginx startup) by using `set $upstream` variables and pointing nginx at Docker's internal resolver (`127.0.0.11`). This prevents nginx from failing to start when upstream containers are not yet healthy.

## Rationale

- **Single config file**: the entire routing setup is readable in one place
- **No label coupling**: services have no knowledge of the reverse proxy
- **Predictable behavior**: nginx semantics are well-documented and widely understood
- **Simpler debugging**: `nginx -t` validates config; no dynamic config watching
- **Same feature set**: everything Traefik provided (CORS, rate limiting, forward auth) is replicated with equivalent nginx directives

## Alternatives Considered

- **Keep Traefik**: familiar to teams using Kubernetes. Adds value at scale with many services and auto-discovery. Unjustified complexity for two upstreams.
- **Caddy**: simpler config syntax, automatic HTTPS. Less battle-tested than nginx for production workloads; smaller ecosystem.

## Consequences

- All routing changes require editing `infra/nginx/nginx.conf` and reloading nginx (`docker exec efir-nginx nginx -s reload`)
- CORS origins must be updated manually in `nginx.conf` (no env var injection without additional tooling)
- In production, TLS termination is handled by nginx — add `listen 443 ssl` blocks and mount certs into the container
- The `infra/traefik/` directory is no longer used and can be removed

# Shared Packages

Reusable Go packages shared across all microservices.

## Packages

| Package                     | Purpose                                   |
| --------------------------- | ----------------------------------------- |
| [errors](errors/)           | Unified gRPC/HTTP error code mapping      |
| [config](config/)           | Shared config structs & environment types |
| [logger](logger/)           | Structured JSON logging with context      |
| [middleware](middleware/)   | gRPC interceptors                         |
| [healthcheck](healthcheck/) | HTTP health/readiness endpoints           |
| [nats](nats/)               | NATS/JetStream connection helpers         |
| [valkey](valkey/)           | Key naming conventions & Lua scripts      |
| [mapper](mapper/)           | Generic slice/map transforms              |
| [testutil](testutil/)       | Testcontainers for integration tests      |

## Import

```go
sharederrors "github.com/sudobytemebaby/efir/services/shared/pkg/errors"
sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
sharedlogger "github.com/sudobytemebaby/efir/services/shared/pkg/logger"
sharedmiddleware "github.com/sudobytemebaby/efir/services/shared/pkg/middleware"
sharedhealth "github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
sharedvalkey "github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
sharedmapper "github.com/sudobytemebaby/efir/services/shared/pkg/mapper"
sharedtestutil "github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
```

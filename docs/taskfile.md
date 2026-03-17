# Taskfile

Efir uses [Task](https://taskfile.dev) as a task runner. Tasks are split across modular files under `tasks/` and included into the root `Taskfile.yml`.

## Installation

```bash
# Arch Linux
pacman -S go-task

# macOS
brew install go-task

# anywhere via Go
go install github.com/go-task/task/v3/cmd/task@latest
```

> On most systems the binary is called `go-task`, not `task`. All examples below use `go-task`. If your system installs it as `task`, use that instead.

---

## Quick start

```bash
# list all available tasks
go-task

# first-time setup after cloning
go-task setup
```

`setup` does the following automatically: creates the Docker network, copies `.env.example → .env` and `docker-compose.dev.example.yml → docker-compose.dev.yml`, configures git hooks, and installs all dev tools.

---

## Tasks reference

### Top-level

| Task            | Description                                                             |
| --------------- | ----------------------------------------------------------------------- |
| `go-task`       | List all available tasks                                                |
| `go-task setup` | Initialize project after clone                                          |
| `go-task tools` | Install pinned dev tools (protoc-gen-go, golangci-lint, mockery, goose) |

> **Note:** `buf` is not installed via `go install`. Install it separately:
> Arch: `yay -S buf` · macOS: `brew install bufbuild/buf/buf` · [other](https://buf.build/docs/installation)

---

### Docker — `docker:`

| Task                                  | Description                                        |
| ------------------------------------- | -------------------------------------------------- |
| `go-task docker:up`                   | Start all services                                 |
| `go-task docker:down`                 | Stop all services                                  |
| `go-task docker:ps`                   | Show container status                              |
| `go-task docker:logs SERVICE=auth`    | Tail logs for a specific service                   |
| `go-task docker:restart SERVICE=auth` | Restart a specific service                         |
| `go-task docker:infra:up`             | Start infra only (postgres, nats, valkey, traefik) |
| `go-task docker:infra:down`           | Stop infra                                         |
| `go-task docker:obs:up`               | Start infra + full observability stack             |
| `go-task docker:obs:down`             | Stop observability stack                           |
| `go-task docker:sidecar:up`           | Start sidecar containers                           |
| `go-task docker:sidecar:down`         | Stop sidecar containers                            |

---

### Go — `go:`

| Task                          | Description                                    |
| ----------------------------- | ---------------------------------------------- |
| `go-task go:test`             | Run tests across all workspace modules         |
| `go-task go:lint`             | Run golangci-lint across all workspace modules |
| `go-task go:generate`         | Regenerate mocks across all workspace modules  |
| `go-task go:run SERVICE=auth` | Run a service locally                          |

---

### Proto — `proto:`

| Task                     | Description                       |
| ------------------------ | --------------------------------- |
| `go-task proto:lint`     | Lint proto files with buf         |
| `go-task proto:generate` | Generate Go code from proto files |

---

### Migrations — `migrate:`

Migrations use [goose](https://github.com/pressly/goose) and are stored in each service's `migrations/` directory. Migration scripts are located in `tasks/scripts/`.

| Task                                            | Description                             |
| ----------------------------------------------- | --------------------------------------- |
| `go-task migrate:up`                            | Apply migrations for all services       |
| `go-task migrate:up SERVICE=auth`               | Apply migrations for a specific service |
| `go-task migrate:down SERVICE=auth`             | Roll back last migration for a service  |
| `go-task migrate:create SERVICE=user NAME=init` | Create a new migration file             |
| `go-task migrate:status`                        | Show migration status for all services  |
| `go-task migrate:status SERVICE=user`           | Show migration status for a service     |

---

## Common workflows

```bash
# start developing: spin up infra, run the auth service locally
go-task docker:infra:up
go-task go:run SERVICE=auth

# before committing
go-task go:lint
go-task go:test

# after editing .proto files
go-task proto:lint
go-task proto:generate

# after pulling changes that include new migrations
go-task migrate:up

# add a migration for the user service
go-task migrate:create SERVICE=user NAME=add_avatar_url
```

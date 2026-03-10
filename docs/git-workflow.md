# Git Workflow

## Branch Naming

Format: `<type>/<short-description-in-kebab-case>`

| Type        | When to use                            | Example                       |
| ----------- | -------------------------------------- | ----------------------------- |
| `feature/`  | New functionality                      | `feature/auth-service`        |
| `fix/`      | Bug fixes                              | `fix/jwt-expiry-validation`   |
| `infra/`    | Infrastructure, docker, configs        | `infra/traefik-ssl-setup`     |
| `refactor/` | Refactoring without behavior change    | `refactor/message-repository` |
| `test/`     | Adding or fixing tests                 | `test/auth-service-unit`      |
| `docs/`     | Documentation, ADRs                    | `docs/adr-jwt-auth`           |
| `chore/`    | Routine: dependencies, code generation | `chore/proto-generation`      |

### Rules

- One branch = one task (one epic or subtask)
- Branches created from `main`, merge back to `main`
- Branch names in English, lowercase, words separated by hyphen
- Delete branch after merge

## Commit Messages

Format: [Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <description>`

### Examples

```
feat(auth): add refresh token rotation
fix(gateway): return 401 on expired jwt instead of 500
infra(postgres): add init scripts for service databases
test(room): cover permission checks in service layer
docs(adr): add decision record for cursor pagination
chore(proto): regenerate go code from updated schemas
refactor(message): extract pagination logic to helper
```

### Rules

- Description in English, lowercase, no period at end
- `scope` — service or component name: `auth`, `gateway`, `infra`, `proto`, `shared`, `sidecar`
- One commit = one logical change
- Don't commit generated files separately — include in the same commit that triggered generation

### Types

- `feat`: New feature
- `fix`: Bug fix
- `infra`: Infrastructure changes
- `refactor`: Code refactoring
- `test`: Adding or fixing tests
- `docs`: Documentation
- `chore`: Maintenance, dependencies

## Workflow

1. Create branch from `main`: `git checkout -b feature/auth-service`
2. Make commits following conventions
3. Push and create PR
4. After merge, delete branch

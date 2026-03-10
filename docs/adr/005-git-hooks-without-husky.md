## ADR-005: Git Hooks Without Husky

## Status
Accepted

## Context
We need to validate commit messages to enforce Conventional Commits format. Traditionally, this is done with Husky (npm package) which manages Git hooks.

## Decision
Use pure bash scripts for local Git hooks without any npm/bun dependencies, and validate the same commit message pattern again in CI.

## Rationale
- **Zero dependencies**: No npm/bun packages needed
- **Husky v9 deprecated**: The `husky install` command is deprecated and will fail in v10.0.0
- **Simple pattern matching**: Using `grep -qE` for validation is fast and reliable
- **Future-proof**: Plain shell scripts will always work regardless of npm ecosystem changes
- **No node_modules**: Keeps the repository clean
- **Same rule in CI**: The repository can enforce the same message format remotely without bringing Node tooling into the Go project

## Implementation

### Hook Setup
```bash
# Configure git to use .githooks directory
git config core.hooksPath .githooks
```

### Commit Message Hook (.githooks/commit-msg)
```bash
#!/bin/sh

# Conventional Commits validator
# Format: <type>(<scope>): <description>
# Example: feat(auth): add jwt validation

commit_msg=$(cat "$1")

# Allowed types: feat, fix, infra, refactor, test, docs, chore
pattern="^(feat|fix|infra|refactor|test|docs|chore)(\([a-z0-9-]+\))?: .{1,100}$"

if ! echo "$commit_msg" | grep -qE "$pattern"; then
  echo "✗ Invalid commit message format"
  echo "Expected: <type>(<scope>): <description>"
  echo "Allowed types: feat, fix, infra, refactor, test, docs, chore"
  exit 1
fi
```

## Alternatives Considered
- **Husky npm package**: Traditional approach, but deprecated, adds unnecessary dependency
- **bunx commitlint**: Works but requires bun installation on each machine
- **Lefthook**: Another npm-based hook manager, still adds dependency
- **CI-only validation**: Would only catch issues in CI, not locally

## Consequences
- Commit messages validated locally on every `git commit`
- Fast validation (pure shell, no external process spawn)
- No npm/bun dependencies in the project
- CI pipeline validates the same regex-based rule (see ADR-006)
- No package.json needed for git hooks

## ADR-005: Git Hooks Without Husky

## Status

Accepted

## Context

We need to validate commit messages to enforce Conventional Commits format. Traditionally, this is done with Husky (npm package) which manages Git hooks.

## Decision

Use plain shell scripts for Git hooks without the Husky npm package.

## Rationale

- **Husky v9 is deprecated**: The `husky install` command is deprecated and will fail in v10.0.0
- **No dependency needed**: Git hooks are just shell scripts — no npm package required
- **Simpler**: Using `bunx commitlint --edit $1` directly in the hook script works perfectly
- **Future-proof**: Plain shell scripts will continue to work regardless of Husky versions
- **Less complexity**: No package.json, no node_modules, no npm/bun dependencies

## Implementation

```sh
# .husky/commit-msg
#!/usr/bin/env sh
bunx --no -- commitlint --edit ${1}
```

This hook runs on every `git commit` and validates the commit message format.

## Alternatives Considered

- **Husky bun npackage**: Traditional approach, but deprecated and adds unnecessary dependency
- **Lefthook**: Another npm-based hook manager, still adds dependency
- **CI-only validation**: Skip local hooks, validate only in CI pipeline

## Consequences

- Commit message validation still works locally (via shell script)
- CI pipeline also validates (see ADR-006 for CI setup)
- No npm/bun dependencies in package.json for git hooks
- Simpler maintenance, no package version tracking

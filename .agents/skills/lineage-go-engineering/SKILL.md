---
name: lineage-go-engineering
description: Use when writing or reviewing Lineage Go code, CLI behavior, runtime package loading, provider adapters, or tests in this repository.
---

# Lineage Go Engineering

Lineage is a local agent package runtime. Keep changes small, deterministic, and provider-neutral unless a file is explicitly an adapter or shim.

## Working Rules

- Preserve the public CLI contract unless the task explicitly changes it.
- Keep reusable source under `internal/` and command entrypoints under `cmd/`.
- Keep core package, config, and runtime logic independent from any one agent provider.
- Put provider-specific behavior behind `internal/provider` or `internal/shim`.
- Prefer deterministic output: sort discovered names, normalize paths, and avoid timestamps in manifests unless required.
- Treat package manifests and package directories as untrusted input.
- Never add behavior that exports secrets, credentials, local auth files, shell history, or private machine state.
- Make operations idempotent where possible: running the same command twice should not duplicate config or corrupt files.

## Validation

Run:

```bash
go test ./...
```

For CLI-facing changes, add or update tests around command output and error behavior.

# Contributing To Lineage

Lineage is early, so the best contributions are focused, testable, and careful about package safety.

## Local Setup

```bash
go test ./...
go run ./cmd/lineage --help
```

## Development Guidelines

- Keep the runtime provider-neutral unless you are working inside an explicit provider boundary.
- Treat package contents as untrusted input.
- Keep manifests deterministic and human-readable.
- Do not add code or docs that encourage sharing secrets, credentials, provider login state, or private machine state.
- Add tests for package discovery, config changes, launch planning, and any safety checks you touch.

## Pull Requests

Use the pull request template and include verification output. For behavior that affects package setup or local files, explain what a receiver can inspect before enabling it.

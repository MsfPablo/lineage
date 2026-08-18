# Testing

Run the full test suite:

```bash
go test ./...
```

Before merging package-format or setup-flow changes, add tests for:

- Deterministic manifest output.
- Idempotent command behavior.
- Package paths that try to escape the package or workspace root.
- Package contents that look like secrets.
- Receiver-side setup prompts that must not run without explicit permission.

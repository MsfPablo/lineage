# Source Layout

Lineage uses the standard Go project layout:

- `cmd/lineage` contains the CLI entrypoint.
- `internal/` contains runtime, package, config, provider, and shim code.

This directory exists so contributors looking for `src/` can find the active source layout without moving Go packages away from idiomatic paths.

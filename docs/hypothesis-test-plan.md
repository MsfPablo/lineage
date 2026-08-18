# Hypothesis And Test Plan

## Hypotheses

- A package can preserve a useful agent environment without requiring users to rewrite it in a new workflow language.
- A receiver can inspect package contents before enabling them.
- Package enablement can be idempotent and safe for repeated runs.
- Provider-specific launch behavior can stay behind explicit adapter boundaries.

## Current Tests

- Project config load, save, and discovery.
- Package initialization, manifest loading, and discovery.
- Provider launch planning.
- Runtime plan rendering.
- Shim creation.

## Needed Tests

- Export/import archive round trip.
- Exact file reconstruction for package contents.
- Secret and ignored-file detection.
- Permission-gated setup flow.
- Path traversal rejection.
- Repeated enablement and repeated setup idempotency.
- Cross-platform path behavior.

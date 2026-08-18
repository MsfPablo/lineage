---
name: lineage-package-safety
description: Use when changing Lineage package manifests, package export/import behavior, setup prompts, file materialization, or secret-safe sharing rules.
---

# Lineage Package Safety

Package receivers should be able to inspect what a package will add before it affects their local agent environment.

## Safety Rules

- Treat every imported package as untrusted until validated.
- Do not execute setup actions during inspect or import.
- Setup actions must be explicit, reviewable, and permission-gated.
- Do not include API keys, auth tokens, `.env` values, local credentials, shell history, or provider login state in packages.
- Data resources such as CSV files, trackers, and small local databases may be package contents only when intentionally included and safe to share.
- Prefer templates or schema files when private source data should not be shared.
- Validate paths before writing files so package contents cannot escape the target workspace.
- Keep manifests deterministic and human-readable.

## Review Questions

- What exactly will the receiver see before enabling this package?
- What files will be created or changed?
- What provider assumptions are embedded?
- Can the same package be enabled twice without damage?

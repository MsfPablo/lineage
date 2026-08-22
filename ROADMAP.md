# Roadmap

This roadmap is intentionally limited to the public local package runtime.

## Done

- Package creation, enablement, and launch planning.
- Manifest schema versioning, export authority, and content digests.
- Secret scanning and path-traversal protection for package-controlled input.
- `lineage package validate`, `lineage package export`, `lineage package import`.
- Registry publish and pull backed by the Lineage website API.
- `lineage add` as the one-command receiver path for published packages.
- GitHub device-flow login, logout, and publisher identity checks.
- `lineage list`, `lineage disable`, `lineage inspect`, and `lineage doctor`.
- Workflow execution through `lineage workflow run`.
- Claude and Codex provider materialization, dry-run previews, and local shims.
- Permission-gated materialization: enabling a package actually stages skills
  into a provider's own directory and generates its context file, with an
  explicit confirmation before anything is written.

## Next

- Finish the active receiver-path hardening work:
  - local `.tgz` archives through `lineage add`;
  - idempotent repeated `add`;
  - package-declared setup files/directories with an explicit creation prompt;
  - provider entrypoints and capabilities surfaced in the registry and package pages.
- Keep public docs synchronized across README, Wiki, Discussions, and the
  website whenever the receiver flow changes.
- Verify the CLI on Windows, including shim generation and binary
  resolution.
- Add an end-to-end integration fixture covering the full author-to-receiver
  flow through the real CLI.
- Add automated drift checks for the bootstrap prompt copy embedded on package
  pages.

## Later

- Design the `.lineage` artifact (project- and user-level state) as a
  deliberate whole, rather than one field at a time.
- Broaden provider adapter coverage while keeping the core package format
  provider-neutral.
- Add stronger capability enforcement if declarative capability visibility is
  not enough for real receiver trust.

---
name: lineage-pr-issue-template
description: Use when creating or reviewing Lineage issues, pull requests, release notes, or contribution guidance so community reports and PRs follow the repository templates.
---

# Lineage PR And Issue Template

Use the repository templates as the source of truth for community contributions.

## Issues

- Bug reports must include: what happened, steps to reproduce, environment, package impact, and confirmation that secrets were removed.
- Feature requests must include: problem, proposed behavior, area, and compatibility notes when relevant.
- Package safety issues must include: concern, sanitized package details, and expected guardrail.
- Contributors should assign an issue to themselves before starting work.

## Pull Requests

Every PR should include:

- A linked assigned issue. PRs without an issue should not be reviewed except maintainer-only housekeeping.
- Summary of what changed and why.
- Package impact checklist.
- Safety checklist.
- Verification commands or a clear reason tests were not run.
- Notes for reviewers when behavior crosses package/provider boundaries.
- `develop` as the base branch unless a maintainer says otherwise.

## Tone

Keep contribution text clear and practical. Do not expose private roadmap details or speculate about unreleased product surfaces.

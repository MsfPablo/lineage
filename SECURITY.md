# Security Policy

Lineage packages can affect local agent environments, so package safety matters.

## Reporting A Vulnerability

Please report private security issues through GitHub Security Advisories. Do not open a public issue with working exploit details, secrets, or private package contents.

## Sensitive Data

Lineage should not package or export:

- API keys, auth tokens, or provider credentials
- `.env` values
- Shell history
- Local credential stores
- Provider login state
- Private machine-local cache files

If a workflow needs configuration, prefer setup prompts, templates, or explicit user-provided values on the receiver's machine.

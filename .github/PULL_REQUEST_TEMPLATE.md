## Summary

What changed, and why?

## Package Impact

- [ ] Package format or manifest behavior
- [ ] Package discovery or enablement
- [ ] Provider launch/shim behavior
- [ ] Setup or permission flow
- [ ] Documentation only
- [ ] Tests only

## Safety

- [ ] This change does not export secrets, credentials, private prompts, or machine-local state.
- [ ] Package inputs are treated as untrusted.
- [ ] Path handling prevents traversal outside the intended workspace/package root where applicable.
- [ ] Provider-specific logic stays behind an explicit boundary.

## Verification

Paste the commands or checks you ran:

```text
go test ./...
```

## Notes For Reviewers

Call out compatibility concerns, follow-up work, or areas that need extra scrutiny.

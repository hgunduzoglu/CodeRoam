<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->
---
name: verify-repository
description: Run CodeRoam's final quality gate and select additional checks from changed paths.
---

# Verify CodeRoam

```bash
pwd
git status --short
git branch --show-current
git diff --name-only
make fmt
make lint
make test
```

If protocol changed:

```bash
buf lint
buf breaking --against '.git#branch=main'
make proto
```

If containers or module files changed:

```bash
make build
```

If mobile/editor/terminal touch behavior changed, update the physical-device matrix; emulator-only
verification is insufficient.

Report commands, pass/fail, skipped checks, and residual risk.

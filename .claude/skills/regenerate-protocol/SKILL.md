<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->
---
name: regenerate-protocol
description: Lint CodeRoam Protobuf schemas, check compatibility, regenerate Go/Dart outputs, and validate consumers.
---

# Regenerate Protocol

```bash
buf lint
buf breaking --against '.git#branch=main'
make proto
git status --short protocol
```

Inspect generated changes, run Go consumers, Flutter analysis/tests, and protocol fixtures. Never
hand-edit or silently discard generated output.

# CodeRoam Agent Guidance Placement

This guidance is already installed in the repository root.

- Codex reads `AGENTS.md`, `.agents/skills`, and `.codex/agents`.
- Claude Code reads `CLAUDE.md`, `.claude/skills`, and `.claude/agents`.
- Nested deployable instructions are present for mobile, control plane, worker, relay, agent,
  protocol, editor, and terminal.

Verify:

```bash
python3 scripts/sync-agent-skills.py --check
```

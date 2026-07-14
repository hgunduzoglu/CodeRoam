# Codex and Claude Code Compatibility

- Canonical project guidance: root/nested `AGENTS.md`.
- Claude wrappers: adjacent `CLAUDE.md` importing `@AGENTS.md`.
- Canonical skills: `.agents/skills/<name>/SKILL.md`.
- Claude mirrors: `.claude/skills/<name>/SKILL.md`.
- Codex agents: `.codex/agents/*.toml`.
- Claude agents: `.claude/agents/*.md`.

After skill changes:

```bash
python3 scripts/sync-agent-skills.py
python3 scripts/sync-agent-skills.py --check
```

Do not commit `CLAUDE.local.md` or `.claude/settings.local.json`.

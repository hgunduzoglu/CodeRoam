<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->
---
name: implement-vertical-slice
description: Implement an approved CodeRoam MVP feature end to end across its owning modules, contracts, mobile surfaces, tests, and documentation.
---

# Implement a Vertical Slice

1. Read product, development spec, and nearest instructions.
2. Identify one owning module and all required consumers.
3. Create an ExecPlan when required by `PLANS.md`.
4. Define observable acceptance behavior.
5. Implement in dependency order:
   - domain/application behavior,
   - persistence/outbox,
   - control-plane contract,
   - data-plane contract only when necessary,
   - Flutter/WebView integration.
6. Add unit, integration, negative, reconnect, and touch tests appropriate to the feature.
7. Use specialists only for disjoint paths.
8. Run focused checks and then the full applicable gate.
9. Review against `docs/agent/code_review.md`.
10. Update docs and durable learnings.

Do not introduce organization/policy scope or advanced offline synchronization.

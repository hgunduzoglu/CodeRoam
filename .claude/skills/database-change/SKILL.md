<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->
---
name: database-change
description: Safely change CodeRoam PostgreSQL schemas, module-owned tables, migrations, transactions, read models, Redis usage, or outbox behavior.
---

# Database Change

1. Identify the owning control-plane module/schema.
2. Confirm no other module writes the table.
3. Define forward migration, compatibility, backfill, recovery, indexes, locks, and transaction/outbox
   effects.
4. Keep authorization inside owning application interfaces.
5. Keep Redis ephemeral.
6. Add migration, repository, transaction, duplicate-delivery, and recovery tests.
7. Run focused integration tests and the full gate.

Do not add organization, membership, role, or generic policy tables in MVP.

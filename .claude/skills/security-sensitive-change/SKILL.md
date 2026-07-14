<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->
---
name: security-sensitive-change
description: Implement and review CodeRoam changes affecting pairing, Noise, tickets, device/agent trust, relay, filesystem, subprocesses, AI permissions, previews, secrets, or runbooks.
---

# Security-Sensitive Change

1. Identify assets, actors, trust boundaries, and attacker-controlled values.
2. State expected authorization and failure behavior before editing.
3. Use established cryptographic libraries; never invent cryptography.
4. Enforce least privilege, explicit validation, expiry, replay protection, timeout, cancellation,
   bounded output, and safe audit metadata.
5. Add negative tests for bypass, stale/revoked state, replay, malformed input, escape, injection,
   secret exposure, and partial failure.
6. Spawn `security-reviewer` after implementation.
7. Resolve proven findings and run the full gate.

Never weaken a control for demo convenience. Never log sensitive payloads. Never grant AI
production authority.

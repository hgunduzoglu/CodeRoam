# CodeRoam Workspace Agent Guidance

- Run non-root and establish an outbound relay connection.
- Generate/protect agent identity and implement secure pairing.
- Constrain all filesystem operations to registered project roots.
- Structured subprocess execution only; never concatenate shell input.
- Apply timeouts, cancellation, process-tree cleanup, bounded output, and sanitized environment.
- Run Git, LSP, Codex, Claude Code, logs, forwarding, and predefined runbooks.
- AI cannot push, deploy, access protected secrets, or execute production runbooks autonomously.
- Add negative tests for escape, injection, replay, revocation, and process leaks.

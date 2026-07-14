# Durable CodeRoam Learnings

Record only non-obvious facts that should affect future agent work.

## 2026-07-14 — Runtime and MVP scope

- The normal runtime path is client ⇄ relay ⇄ agent; SSH tunneling is not an intermediate product
  path.
- Touch UX validation is Milestone 0.
- Organization/RBAC/policy scope is outside MVP.
- Offline support is explicit Offline Drafts, not advanced synchronization.
- Root `go test ./...` is invalid for this multi-module workspace; use `make test-go`.

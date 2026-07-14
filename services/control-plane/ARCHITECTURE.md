# Control Plane Architecture

CodeRoam's control plane is a modular monolith.

Modules and owned schemas:

- `auth`
- `device`
- `workspace`
- `session`
- `preview`
- `integration`
- `runbook`
- `outbox`

There is deliberately no organization, team, membership, enterprise RBAC, or generic policy module
in the MVP. Every resource is directly owned by one user.

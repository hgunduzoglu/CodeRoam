# CodeRoam Control Plane Guidance

- Modular monolith; each module owns one PostgreSQL schema.
- Modules communicate through Go application interfaces.
- No organization, team, membership, enterprise RBAC, or generic policy engine in MVP.
- Authorization is explicit single-owner resource validation plus device/agent status.
- Cross-schema read models never decide authorization or trust.
- Durable external side effects use transactional outbox.
- Pairing/session tickets are short-lived metadata, never E2E keys.
- REST handlers validate transport input and call application services.

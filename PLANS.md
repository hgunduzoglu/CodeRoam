# CodeRoam ExecPlans

Use an ExecPlan when work:

- spans three or more deployables,
- changes pairing, cryptography, relay, Protobuf, or resume semantics,
- changes PostgreSQL ownership or migrations,
- changes filesystem/process/runbook trust boundaries,
- introduces a multi-milestone feature,
- or is explicitly requested as a plan.

Store plans in `docs/plans/<YYYY-MM-DD>-<name>.md`.

Required sections:

```markdown
# Title
## Objective
## Scope
## Current state
## Design
## Milestones
## Progress
## Decisions
## Validation
## Recovery and rollback
## Open risks
```

Plans are living documents. Update progress and decisions as facts change. A milestone is complete
only after its verification succeeds.

## What changed

Describe observable CodeRoam behavior.

## Scope

- [ ] Mobile/touch UX
- [ ] Control plane
- [ ] Worker
- [ ] Relay
- [ ] Workspace agent
- [ ] Protocol/pairing
- [ ] Editor
- [ ] Terminal
- [ ] Deployment/docs

## Architecture and security

- Owning module/schema:
- Pairing/session impact:
- Trust boundaries:
- ExecPlan/ADR:
- [ ] Runtime remains client ⇄ relay ⇄ agent
- [ ] Relay remains payload-opaque
- [ ] No organization/policy scope added
- [ ] Offline Drafts cannot silently overwrite changed remote content
- [ ] No secret or engineering payload logging
- [ ] AI cannot push, deploy, or run production actions autonomously

## Verification

List exact commands and results.

- [ ] Focused tests
- [ ] `make fmt`
- [ ] `make lint`
- [ ] `make test`
- [ ] Physical-device touch matrix updated when applicable

## Unverified

State checks that were not run.

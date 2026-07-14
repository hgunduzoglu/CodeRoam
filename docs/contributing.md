# Contributing to CodeRoam

1. Read `PRODUCT.md`, `docs/development-spec.md`, and the applicable `AGENTS.md`.
2. Create a branch from `main`.
3. Keep changes within one owning module unless a documented vertical slice requires more.
4. Add tests for behavior and failure cases.
5. Run:

```bash
make fmt
make lint
make test
```

6. Use Conventional Commits.
7. Do not commit secrets, engineering payloads, generated credentials, or private workspace data.

# ModelRails

ModelRails is a deterministic, fail-closed decision engine for model actions. It evaluates explicit, version-controlled invariants before allowing deployments, upgrades, capability changes, cross-model interactions, or environment promotions. ModelRails is CLI-first and designed for CI/CD integration.

## Scope (v1)

ModelRails v1 is intentionally narrow:
- Pre-flight evaluation only (no runtime enforcement)
- Read-only inputs (no mutation of systems)
- Deterministic decisions from explicit policy

It does not train or evaluate models, manage infrastructure, or run as a long-lived service.

## Why it exists

Most production ML failures come from ungoverned change: capability drift, unsafe interactions, and undocumented behavior shifts. Metrics do not prevent these failures. ModelRails enforces invariants that must hold before changes proceed.

## Core principles

- **Deterministic**: identical inputs produce identical outputs
- **Fail-closed**: invalid or missing inputs block the action
- **Explicit**: no inference, no hidden defaults
- **Auditable**: structured evidence for every non-ALLOW decision
- **Separation of concerns**: metadata collection, evaluation, aggregation, presentation

## What ModelRails evaluates

Model actions (v1):
- Deploy new model versions
- Upgrade dependencies
- Connect model-to-model interactions
- Promote environments

Invariant types (v1):
- Capability subsets
- Allowlist / denylist interactions
- Version pins
- Environment gates
- Resource bounds

## CLI usage

```
modelrails check \
  --spec invariants.yaml \
  --action action.yaml \
  --metadata metadata.yaml
```

Exit codes:
- ALLOW → 0
- WARN → 0
- BLOCK → 1
- ERROR → 2

## Repo structure

```
.
├── cmd/
├── internal/
├── docs/
│   └── tech/
│       └── modelrails-v1.md
├── examples/
└── README.md
```

## Design reference

Authoritative design and scope are defined in [docs/tech/modelrails-v1.md](docs/tech/modelrails-v1.md).

## Contributing

This project is opinionated by design. Before proposing changes:
1. Read [docs/tech/modelrails-v1.md](docs/tech/modelrails-v1.md)
2. Keep v1 scope and determinism requirements intact
3. Prefer design discussion before implementation changes

## License

Apache 2.0

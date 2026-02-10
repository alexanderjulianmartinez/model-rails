# ModelRails

**ModelRails** is an infrastructure framework for enforcing **hard invariants, safety rails, and lifecycle controls** around ML/AI models in production.

It sits *around* model serving and training systems — not inside them — acting as a **control plane** that decides *whether* a model or change is allowed to run, deploy, or interact with other systems.

Think:
> *“Kubernetes admission control, but for models.”*

## Why ModelRails Exists

As ML systems mature, failures stop being about accuracy and start being about:
- Silent regressions
- Unsafe model interactions
- Undocumented behavior drift
- Models operating outside their intended domain
- Human trust erosion

Metrics alone don’t prevent these failures.

**ModelRails enforces rules that must never be violated**, regardless of model performance.

## Core Principles

- **Invariants over metrics**  
  Some things must never happen, even if accuracy improves.

- **Fail closed by default**  
  If a rule can’t be evaluated, the action is blocked.

- **Model-agnostic**  
  Works with LLMs, classical ML, fine-tuned models, and ensembles.

- **Explicit intent**  
  Every allowed behavior must be declared, versioned, and auditable.

- **Infrastructure-first**  
  This is not a prompt library or evaluation toolkit.

## What ModelRails Does (v1)

ModelRails v1 focuses on **decision-time enforcement**, not training.

### Enforced Actions
- Model deployment
- Model version upgrades
- Prompt or input class changes
- Model-to-model calls
- Production traffic enablement

### Guardrail Types
- **Schema invariants** (inputs/outputs must conform)
- **Capability constraints** (what a model is allowed to do)
- **Interaction constraints** (which models can talk to which)
- **Change safety rules** (what changed, and whether it’s allowed)
- **Blast-radius limits** (scope of impact)

## What ModelRails Is Not

- A training framework  
- A prompt engineering toolkit  
- A metrics dashboard  
- An A/B testing platform  
- A policy engine for humans  

ModelRails governs **models**, not people.

## High-Level Architecture
``` yaml
Request / Deployment / Change -> ModelRails (Invariant Engine) -> ALLOW BLOCK -> Proceed Emit Reason
```

ModelRails is typically invoked by:
- CI/CD pipelines
- Model registries
- Serving gateways
- Orchestration systems

## Example Use Cases

- Prevent deploying a model that introduces a new output field without approval
- Block a model from calling another model outside its trust domain
- Require explicit sign-off for expanding a model’s capability set
- Detect unsafe behavioral changes even when metrics improve
- Enforce “no self-modifying prompts” policies

## Project Status

🚧 **Early design & v1 implementation**

Current focus:
- Invariant specification format
- Decision engine
- Integration hooks (CI / deploy-time)
- Clear non-goals

See the full design in:
``` yaml
/docs/tech/modelrails-v1.md
```

## Repo Structure
``` yaml
.
├── cmd/
├── internal/
├── docs/
│ └── tech/
│ └── modelrails-v1.md
├── examples/
└── README.md
```
## Long-Term Vision

ModelRails is designed to grow into:
- A shared control plane for multi-model systems
- A foundation for safe autonomous agents
- A critical layer in ML/AI platform stacks
- A prerequisite for scaling trust in AI systems

## Contributing

This project is intentionally opinionated.
Design discussions matter more than code volume.

Before contributing:
1. Read `/docs/tech/modelrails-v1.md`
2. Understand the invariants-first philosophy
3. Propose changes via design discussion, not just PRs

## License

Apache 2.0 

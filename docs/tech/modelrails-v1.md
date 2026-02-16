# ModelRails v1

> This document provides authoritative context for AI-assisted development of ModelRails v1.
> All AI prompts in VS Code should reference this file.
> Scope discipline is critical. ModelRails v1 is intentionally narrow.

---

# 1. Overview

ModelRails is a lightweight **control plane for model-related actions**.

It evaluates **explicit invariants** before allowing model operations such as:

- Model deployment
- Model version upgrades
- Capability expansion
- Model-to-model interaction
- Environment promotion

ModelRails v1 is:

- Deterministic
- Fail-closed
- Read-only
- CLI-first
- Infra-focused (not ML-research-focused)

---

# 2. Core Problem

Modern ML systems lack **explicit guardrails** for:

- Capability drift
- Unsafe deployment changes
- Cross-model interaction risk
- Silent expansion of behavior
- Undocumented version changes

Most organizations rely on:
- Human review
- Informal documentation
- Monitoring after the fact

ModelRails shifts this left:

> “Prove this action satisfies declared invariants before execution.”

---

# 3. V1 Scope

ModelRails v1 only evaluates **model action decisions**.

It does NOT:
- Train models
- Tune models
- Evaluate model quality
- Modify models
- Perform runtime enforcement
- Integrate deeply with GPUs
- Perform ML research

It is a **pre-flight decision engine**.

---

# 4. Key Concepts

## 4.1 Model Action

A model action is something that may change system behavior.

Examples:
- Deploy new model version
- Upgrade dependency model
- Enable new capability flag
- Connect model A → model B
- Promote staging → production

All actions are evaluated before execution.

---

## 4.2 Invariants

An invariant is a declared condition that must hold.

Each invariant includes:

- Name
- Scope (model, version, interaction, environment)
- Evaluation logic
- Severity (ALLOW, WARN, BLOCK)
- Evidence output

Invariants must be:
- Explicit
- Deterministic
- Auditable
- Version-controlled

---

## 4.3 Decision Model

Every evaluation returns:
Decision:
status: ALLOW | WARN | BLOCK
violated_invariants: []
reasoning: structured explanation


Decision semantics:

- ALLOW → exit code 0
- WARN → exit code 0 (with warning)
- BLOCK → exit code 1

Fail-closed behavior is mandatory.

---

# 5. Architectural Principles

## 5.1 Separation of Concerns

- Metadata collection
- Invariant evaluation
- Decision aggregation
- CLI presentation

These must remain decoupled.

---

## 5.2 Determinism

Given:
- Same invariant spec
- Same metadata
- Same action context

The output must be identical.

No randomness.
No hidden state.

---

## 5.3 Auditability

All decisions must:

- Include evidence
- Include violated invariant names
- Be reproducible
- Be log-friendly
- Be CI-compatible

---

## 5.4 Explicit Over Implicit

ModelRails must never:
- Infer capabilities silently
- Guess policy intent
- Autocorrect unsafe actions
- Mutate system state

---

# 6. Invariant Specification Format

V1 uses YAML-based invariant specs.

Example:

```yaml
version: 1
invariants:
  - name: restrict_capability_expansion
    scope: model_version
    severity: block
    condition:
      type: capability_subset
      allowed_capabilities:
        - summarization
        - classification

  - name: prevent_unapproved_interaction
    scope: interaction
    severity: block
    condition:
      type: allowlist
      allowed_targets:
        - embedding-service
        - retrieval-service
```

# 7. ActionContext Model

The `ActionContext` represents the **specific operation being evaluated**.

It is immutable input to the decision engine.

## 7.1 Purpose

- Define *what* is changing
- Define *where* it is changing
- Define *who* initiated the change
- Provide sufficient context for invariant evaluation

ModelRails never mutates or enriches this object implicitly.

---

## 7.2 Required Fields

```yaml
action_type: deploy | upgrade | connect | promote
model_name: string
model_version: string
environment: dev | staging | production
initiator: string
timestamp: RFC3339 string
metadata_snapshot_ref: string
```

## 7.3 Design Constraints

The `ActionContext` must satisfy the following constraints:

- Must be serializable (JSON or YAML).
- Must be loggable without transformation.
- Must not contain executable logic.
- Must not rely on implicit environment state.
- Must be treated as immutable during evaluation.

All invariants evaluate strictly against this context and supplied metadata.

---

# 8. Model Metadata Model

Model metadata represents **observed, factual system state** at evaluation time.

It is read-only input to the decision engine.

## 8.1 Example Metadata Structure

```yaml
model_name: recommendation-model
model_version: v3.2.1
declared_capabilities:
  - ranking
  - personalization
dependency_models:
  - name: embedding-service
    version: v2.0.0
resource_requirements:
  cpu: "4"
  memory: "16Gi"
  gpu: "1"
trust_domain: user-facing
deployment_environment: production
```

## 8.2 Metadata Rules

Model metadata must follow strict rules to preserve determinism and safety:

- Metadata must reflect actual system state at evaluation time.
- Metadata must not be inferred implicitly.
- Missing required metadata must cause evaluation failure.
- Metadata collection must be independent from invariant evaluation.
- The decision engine must never mutate metadata.
- Partial metadata is treated as invalid input.

If required metadata is missing or malformed, evaluation must fail closed with an ERROR.

---

# 9. Decision Engine

The Decision Engine is the deterministic core of ModelRails.

It performs the following ordered steps:

1. Validate invariant specification.
2. Validate ActionContext.
3. Validate metadata.
4. Evaluate each invariant independently.
5. Aggregate severities.
6. Produce structured decision output.
7. Exit with appropriate status code.

The engine must be side-effect free.

---

## 9.1 Deterministic Aggregation Rules

Severity precedence:

BLOCK > WARN > ALLOW

Aggregation rules:

- If any invariant evaluates to BLOCK, the final decision is BLOCK.
- If no BLOCK exists but at least one WARN exists, the final decision is WARN.
- If all invariants evaluate to ALLOW, the final decision is ALLOW.
- Invariant evaluation order must not affect final outcome.

---

## 9.2 Decision Output Structure

Example output:

```yaml
decision: BLOCK
evaluated_at: 2026-02-16T12:00:00Z
action:
  model_name: recommendation-model
  model_version: v3.2.1
  action_type: deploy
violations:
  - invariant_name: restrict_capability_expansion
    severity: BLOCK
    reason: "Declared capability 'generation' exceeds allowed set."
    evidence:
      allowed_capabilities:
        - ranking
        - personalization
      observed_capabilities:
        - ranking
        - personalization
        - generation
```

The decision output must be structured, deterministic, and auditable.

It must include:

- Final decision status (ALLOW, WARN, or BLOCK)
- Evaluation timestamp
- Action summary (model name, version, action type)
- List of violated invariants (if any)
- Severity for each violation
- Clear human-readable reasoning
- Structured evidence supporting each violation

Output requirements:

- Must be reproducible given identical inputs.
- Must include invariant names.
- Must include explicit reasoning for every non-ALLOW result.
- Must include structured evidence sufficient for debugging.
- Must avoid non-deterministic ordering.
- Must support machine-readable serialization (e.g., JSON).

The decision output is part of the audit surface and must be treated as a stable interface.

---

# 10. CLI Requirements

ModelRails v1 is CLI-first.

Primary command:
```bash
modelrails check \
  --spec invariants.yaml \
  --action action.yaml \
  --metadata metadata.yaml
```

It must support evaluation of:

- Invariant specification
- ActionContext
- Model metadata

The CLI must be suitable for CI/CD integration.

---

## 10.1 CLI Behavior Requirements

The CLI must:

- Produce deterministic output formatting.
- Default to human-readable output.
- Support a machine-readable output mode.
- Validate all inputs before evaluation.
- Provide clear validation errors.
- Avoid implicit defaults for required fields.
- Avoid reliance on environment variables or hidden state.
- Fail fast on invalid input.

All input validation must occur before invariant evaluation begins.

---

## 10.2 Exit Code Semantics

Exit codes must be stable and predictable:

- ALLOW returns exit code 0.
- WARN returns exit code 0.
- BLOCK returns exit code 1.
- ERROR returns exit code 2.

ERROR includes:

- Invalid invariant specification.
- Invalid ActionContext.
- Invalid metadata.
- Unsupported invariant types.
- Evaluation runtime failures.

All ERROR conditions must fail closed.

---

## 10.3 Example Scenarios (ModelRails v1)

Each scenario includes a realistic invariant spec and the expected decision outcome.

### Scenario 1: Blocking a model deployment

Invariant spec:

```yaml
version: 1
invariants:
  - name: production_resource_bounds
    scope: model
    severity: block
    intent: "Block deployments that exceed production resource limits."
    condition:
      type: resource_bounds
      parameters:
        max_cpu: "8"
        max_memory: "32Gi"
        max_gpu: "1"
```

Expected decision: BLOCK (exit code 1) because observed resource requirements exceed the declared bounds.

### Scenario 2: Warning on capability expansion

Invariant spec:

```yaml
version: 1
invariants:
  - name: capability_expansion_review
    scope: version
    severity: warn
    intent: "Warn on capability expansion beyond approved set."
    condition:
      type: capability_subset
      parameters:
        allowed_capabilities:
          - summarization
          - classification
```

Expected decision: WARN (exit code 0) because declared capabilities include items outside the allowed set.

### Scenario 3: Blocking unsafe model interaction

Invariant spec:

```yaml
version: 1
invariants:
  - name: block_unapproved_dependencies
    scope: interaction
    severity: block
    intent: "Block connections to disallowed dependency models."
    condition:
      type: denylist
      parameters:
        denied_targets:
          - external-chat-service
          - untrusted-retrieval
```

Expected decision: BLOCK (exit code 1) when observed dependency targets include any denied model.

---

# 11. Non-Goals (Strictly Enforced)

ModelRails v1 will NOT:

- Perform runtime enforcement.
- Modify infrastructure state.
- Train or evaluate models.
- Detect hallucinations.
- Score model performance.
- Automatically remediate violations.
- Act as a Kubernetes admission controller.
- Implement distributed coordination.
- Introduce adaptive or self-learning policy behavior.
- Perform benchmarking or performance analysis.

Any proposal introducing these features is out of scope for v1.

---

# 12. Failure Modes

The system must explicitly handle:

- Invalid input formats.
- Unknown invariant types.
- Missing required metadata fields.
- Unsupported action types.
- Conflicting invariant definitions.
- Partial invariant evaluation.
- Spec version mismatches.

Failure handling requirements:

- Produce explicit and actionable error messages.
- Identify the source of failure clearly.
- Exit with ERROR status.
- Never silently degrade behavior.
- Never auto-allow due to ambiguity.

Ambiguity must result in failure.

---

# 13. Design Tradeoffs

## 13.1 CLI Over Service

V1 is CLI-based to ensure:

- Easier auditability.
- No persistent hidden state.
- Clean CI integration.
- Reduced operational complexity.
- Minimal blast radius.

A long-running service introduces state management and distributed coordination complexity and is intentionally deferred.

---

## 13.2 Static Invariants Over Dynamic Learning

Policies are explicitly declared and version-controlled.

There is:

- No heuristic inference.
- No ML-based policy evaluation.
- No adaptive behavior.
- No dynamic policy mutation.

Explicit declarations are preferred over inferred behavior.

---

## 13.3 Determinism Over Flexibility

Given identical:

- Invariant specification
- ActionContext
- Metadata

The output must be identical.

There must be:

- No randomness.
- No implicit time-based logic.
- No non-deterministic ordering.
- No external network dependencies during evaluation.

Determinism is a hard requirement.

---

## 13.4 Explicit Policy Over Implicit Defaults

If a constraint is not declared, it does not exist.

ModelRails must not:

- Assume safe defaults.
- Infer allowed capabilities.
- Apply hidden fallback rules.
- Auto-approve missing constraints.

Ambiguity must result in ERROR, not ALLOW.

---

## 13.5 Design Rationale (Tradeoffs and Rejected Alternatives)

ModelRails v1 optimizes for determinism, auditability, and CI compatibility over flexibility. This drives a CLI-first architecture, explicit policy definitions, and strict validation of inputs.

Key rejected alternatives:
- **Runtime enforcement or admission control**: deferred to avoid persistent state and operational coupling.
- **Adaptive or learning-based policy**: rejected due to non-determinism and audit complexity.
- **Implicit defaults or auto-correction**: rejected to prevent silent policy drift and ambiguous approvals.
- **Automatic remediation**: out of scope for v1; decisions remain read-only.

---

# 14. Future Direction (Explicitly Out of Scope for v1)

Potential future extensions include:

- Multi-model dependency graph invariants.
- Cross-team trust boundary enforcement.
- Policy version migration frameworks.
- Historical decision persistence.
- Admission controller integration.
- Runtime policy sidecars.
- Organization-wide capability registries.
- Distributed evaluation coordination.

These future directions must not influence v1 implementation decisions.

---

# 15. Engineering Standards

Implementation must adhere to:

- Language: Go (preferred) or Python (acceptable).
- Strong typing.
- Unit tests for invariant evaluation logic.
- No global mutable state.
- Clear package or module boundaries.
- Deterministic output formatting.
- Minimal external dependencies.
- Explicit error handling.

Core logic must be independently testable without invoking the CLI.

---

# 16. AI Usage Rules in VS Code

When using AI assistance:

1. Always reference this document explicitly.
2. Do not expand scope beyond v1.
3. Do not introduce ML evaluation features.
4. Preserve deterministic behavior.
5. Fail closed on uncertainty.
6. Maintain separation of concerns.
7. Avoid introducing hidden state or background services.
8. Prefer simple, explicit implementations over abstraction-heavy designs.

AI suggestions that violate scope or determinism must be rejected.

Example prompt:
```bash
Using docs/tech/modelrails-v1.md as authoritative context,
implement invariant evaluation with deterministic aggregation
and fail-closed semantics.
```

---

# 17. V1 Success Criteria

ModelRails v1 is successful if:

- Unsafe capability expansion is blocked.
- Unauthorized model interactions are blocked.
- Decisions are reproducible.
- Output integrates cleanly into CI pipelines.
- Engineers can understand and explain why a decision was made.
- Evaluation behavior is deterministic and auditable.
- There is no hidden state or implicit behavior.

---

# 18. Summary

ModelRails v1 is a deterministic pre-flight decision engine for model-related actions.

It enforces explicit invariants.
It produces auditable decisions.
It fails closed.
It does not enforce at runtime.
It does not learn or adapt.

Its purpose is singular:

Make unsafe model actions impossible to ignore.

package engine

import (
	"strings"
	"testing"

	"github.com/alexanderjulianmartinez/model-rails/internal/spec"
)

func validAction() ActionContext {
	return ActionContext{
		ActionType:          ActionDeploy,
		ModelName:           "rec-model",
		ModelVersion:        "v1.2.3",
		Environment:         EnvDev,
		Initiator:           "ci-bot",
		Timestamp:           "2026-02-16T12:00:00Z",
		MetadataSnapshotRef: "snapshot://abc123",
	}
}

func validMetadata() ModelMetadata {
	return ModelMetadata{
		ModelName:    "rec-model",
		ModelVersion: "v1.2.3",
		DeclaredCapabilities: []string{
			"classification",
		},
		DependencyModels: []DependencyModel{
			{Name: "embedding-service", Version: "v2.0.0"},
		},
		ResourceRequirements: ResourceRequirements{
			CPU:    "2",
			Memory: "4Gi",
			GPU:    "0",
		},
		TrustDomain:           "user-facing",
		DeploymentEnvironment: "dev",
	}
}

func makeSpec(invariants ...spec.Invariant) spec.InvariantSpec {
	return spec.InvariantSpec{
		Version:    spec.SpecVersionV1,
		Invariants: invariants,
	}
}

func makeInvariant(name string, severity string, conditionType string, params map[string]interface{}) spec.Invariant {
	return spec.Invariant{
		Name:     name,
		Scope:    spec.ScopeVersion,
		Severity: severity,
		Intent:   "test invariant",
		Condition: spec.Condition{
			Type:       conditionType,
			Parameters: params,
		},
	}
}

func TestEvaluateAllow(t *testing.T) {
	s := makeSpec(makeInvariant(
		"capability_subset",
		spec.SeverityBlock,
		spec.ConditionCapabilitySubset,
		map[string]interface{}{"allowed_capabilities": []string{"classification"}},
	))

	result, err := Evaluate(s, validAction(), validMetadata())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Decision != DecisionAllow {
		t.Fatalf("expected decision %s, got %s", DecisionAllow, result.Decision)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(result.Violations))
	}
	if result.Explanation != "All invariants satisfied." {
		t.Fatalf("unexpected explanation: %s", result.Explanation)
	}
}

func TestEvaluateWarn(t *testing.T) {
	s := makeSpec(makeInvariant(
		"interaction_allowlist",
		spec.SeverityWarn,
		spec.ConditionAllowlist,
		map[string]interface{}{"allowed_targets": []string{"retrieval-service"}},
	))

	result, err := Evaluate(s, validAction(), validMetadata())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Decision != DecisionWarn {
		t.Fatalf("expected decision %s, got %s", DecisionWarn, result.Decision)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Severity != "WARN" {
		t.Fatalf("expected violation severity WARN, got %s", result.Violations[0].Severity)
	}
	if result.Violations[0].Remediation == "" {
		t.Fatalf("expected remediation, got empty")
	}
	if !strings.Contains(result.Explanation, "violation[1]") {
		t.Fatalf("expected WARN explanation, got %s", result.Explanation)
	}
}

func TestEvaluateBlockPrecedence(t *testing.T) {
	blockInv := makeInvariant(
		"a_block",
		spec.SeverityBlock,
		spec.ConditionCapabilitySubset,
		map[string]interface{}{"allowed_capabilities": []string{"summarization"}},
	)
	warnInv := makeInvariant(
		"b_warn",
		spec.SeverityWarn,
		spec.ConditionAllowlist,
		map[string]interface{}{"allowed_targets": []string{"retrieval-service"}},
	)

	result, err := Evaluate(makeSpec(blockInv, warnInv), validAction(), validMetadata())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Decision != DecisionBlock {
		t.Fatalf("expected decision %s, got %s", DecisionBlock, result.Decision)
	}
	if len(result.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(result.Violations))
	}
	if result.Violations[0].InvariantName != "a_block" {
		t.Fatalf("expected violations sorted by name, got %s", result.Violations[0].InvariantName)
	}
}

func TestEvaluateInvalidAction(t *testing.T) {
	action := validAction()
	action.Timestamp = "invalid"

	s := makeSpec(makeInvariant(
		"capability_subset",
		spec.SeverityBlock,
		spec.ConditionCapabilitySubset,
		map[string]interface{}{"allowed_capabilities": []string{"classification"}},
	))

	_, err := Evaluate(s, action, validMetadata())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "action.timestamp") {
		t.Fatalf("expected timestamp error, got %v", err)
	}
}

func TestEvaluateMissingMetadata(t *testing.T) {
	s := makeSpec(makeInvariant(
		"resource_bounds",
		spec.SeverityBlock,
		spec.ConditionResourceBounds,
		map[string]interface{}{
			"max_cpu":    "2",
			"max_memory": "4Gi",
			"max_gpu":    "0",
		},
	))

	metadata := validMetadata()
	metadata.ResourceRequirements = ResourceRequirements{}

	_, err := Evaluate(s, validAction(), metadata)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "metadata.resource_requirements.cpu") {
		t.Fatalf("expected resource requirements error, got %v", err)
	}
}

func TestActionContextValidate(t *testing.T) {
	valid := validAction()
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	invalid := validAction()
	invalid.ActionType = ""
	invalid.ModelName = ""
	invalid.ModelVersion = ""
	invalid.Environment = "bad-env"
	invalid.Initiator = ""
	invalid.Timestamp = "not-a-time"
	invalid.MetadataSnapshotRef = ""

	err := invalid.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	checks := []string{
		"action.action_type",
		"action.model_name",
		"action.model_version",
		"action.environment",
		"action.initiator",
		"action.timestamp",
		"action.metadata_snapshot_ref",
	}

	for _, needle := range checks {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("expected error to contain %s, got %v", needle, err)
		}
	}
}

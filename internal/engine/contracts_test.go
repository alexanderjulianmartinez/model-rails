package engine

import (
	"testing"

	"github.com/alexanderjulianmartinez/model-rails/internal/spec"
)

type stubFormatter struct {
	value string
}

func (f stubFormatter) Format(result DecisionResult) string {
	return f.value
}

func TestBuildDecisionOutputDefault(t *testing.T) {
	result := DecisionResult{
		Decision:    DecisionAllow,
		Explanation: "ok",
	}

	output := BuildDecisionOutput(result, nil)
	if output != "ok" {
		t.Fatalf("expected default output to be ok, got %s", output)
	}
}

func TestBuildDecisionOutputCustom(t *testing.T) {
	result := DecisionResult{
		Decision:    DecisionAllow,
		Explanation: "ignored",
	}

	output := BuildDecisionOutput(result, stubFormatter{value: "custom"})
	if output != "custom" {
		t.Fatalf("expected custom output, got %s", output)
	}
}

func TestDefaultEvaluatorEvaluate(t *testing.T) {
	s := makeSpec(makeInvariant(
		"capability_subset",
		spec.SeverityBlock,
		spec.ConditionCapabilitySubset,
		map[string]interface{}{"allowed_capabilities": []string{"classification"}},
	))

	e := DefaultEvaluator{}
	result, err := e.Evaluate(s, validAction(), validMetadata())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Decision != DecisionAllow {
		t.Fatalf("expected decision %s, got %s", DecisionAllow, result.Decision)
	}
}

package engine

import "github.com/alexanderjulianmartinez/model-rails/internal/spec"

// MetadataCollector is responsible for assembling observed system metadata
// prior to any invariant evaluation. It must be deterministic and side-effect free.

type MetadataCollector interface {
	Collect(action ActionContext) (ModelMetadata, error)
}

// Evaluator performs invariant evaluation against an action and metadata.
type Evaluator interface {
	Evaluate(specDoc spec.InvariantSpec, action ActionContext, metadata ModelMetadata) (DecisionResult, error)
}

// DefaultEvaluator calls the core evaluation function.
type DefaultEvaluator struct{}

func (e DefaultEvaluator) Evaluate(specDoc spec.InvariantSpec, action ActionContext, metadata ModelMetadata) (DecisionResult, error) {
	return Evaluate(specDoc, action, metadata)
}

// DecisionFormatter renders a DecisionResult for CI/log consumption.
type DecisionFormatter interface {
	Format(result DecisionResult) string
}

// CIFormatter renders CI-friendly output based on the decision explanation.
type CIFormatter struct{}

func (f CIFormatter) Format(result DecisionResult) string {
	return result.Explanation
}

// BuildDecisionOutput renders a decision using the provided formatter.
func BuildDecisionOutput(result DecisionResult, formatter DecisionFormatter) string {
	if formatter == nil {
		formatter = CIFormatter{}
	}
	return formatter.Format(result)
}
